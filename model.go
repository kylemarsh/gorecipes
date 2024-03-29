package main

import (
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var db *sqlx.DB

/*********
 * TYPES *
 *********/

/*User - notion of who can see the recipes*/
type User struct {
	ID                int `db:"user_id"`
	Username          string
	HashedPassword    string `db:"password"`
	PlaintextPassword string `db:"plaintext_pw_bootstrapping_only"`
}

/*Recipe - basic unit of the recipe database */
type Recipe struct {
	ID         int `db:"recipe_id"`
	Title      string
	Body       string `db:"recipe_body"`
	Time       int    `db:"total_time"`
	ActiveTime int    `db:"active_time"`
	Deleted    bool
	New        bool
	Labels     []Label
	Notes      []Note
}

/*Label - a taxonomic tab for recipes */
type Label struct {
	ID    int `db:"label_id"`
	Label string
}

/*Note - a note attached to a recipe */
type Note struct {
	ID       int `db:"note_id"`
	RecipeId int `db:"recipe_id"`
	Created  int `db:"create_date"`
	Note     string
	Flagged  bool
}

/*************
 * FUNCTIONS *
 *************/
// Load //
func activeRecipes(includeBody bool) ([]Recipe, error) {
	var recipes []Recipe
	var q string
	if includeBody {
		q = "SELECT * FROM recipe WHERE deleted = 0"
		// TODO can we populate the labels and recipes at the same time?
		//q = "SELECT recipe.*, label.* FROM recipe join recipe_label using(recipe_id) join label using(label_id)"
	} else {
		q = "SELECT recipe_id, title, total_time, active_time FROM recipe WHERE deleted = 0"
	}
	connect()
	err := db.Select(&recipes, q)
	if err != nil {
		return recipes, err
	}

	var savedErr error
	//TODO load in labels for each recipe
	for i, recipe := range recipes {
		labels, err := labelsByRecipeID(recipe.ID)
		if err != nil {
			savedErr = err
			fmt.Println("error loading labels for recipe", recipe.ID, err)
		}
		recipe.Labels = labels
		recipes[i] = recipe
	}
	return recipes, savedErr
}

func recipeByID(id int, wantLabels bool) (Recipe, error) {
	var recipe Recipe
	var labels []Label
	q := "SELECT * FROM recipe WHERE recipe_id = ?"

	connect()
	err := db.Get(&recipe, q, id)
	if wantLabels == true && err == nil {
		labels, err = labelsByRecipeID(id)
		recipe.Labels = labels
	}
	return recipe, err
}

func labelByID(id int) (Label, error) {
	var label Label
	q := "SELECT * FROM label WHERE label_id = ?"

	connect()
	err := db.Get(&label, q, id)
	return label, err
}

func labelByName(name string) (Label, error) {
	var label Label
	q := "SELECT * FROM label WHERE label = ?"

	connect()
	err := db.Get(&label, q, name)
	return label, err
}

func labelsByRecipeID(id int) ([]Label, error) {
	var labels []Label
	q := "SELECT label.* FROM label join recipe_label using(label_id) WHERE recipe_id = ?"

	connect()
	err := db.Select(&labels, q, id)
	return labels, err
}

func getNoteByID(id int) (Note, error) {
	note := Note{}
	q := "SELECT * FROM note WHERE note_id = ?"

	connect()
	err := db.Get(&note, q, id)
	return note, err
}

func notesByRecipeID(recipe_id int) ([]Note, error) {
	var notes []Note
	q := "SELECT * FROM note WHERE recipe_id = ?"

	connect()
	err := db.Select(&notes, q, recipe_id)
	return notes, err
}

func userByName(username string) (User, error) {
	var user User
	q := "SELECT * FROM user WHERE username = ?"
	connect()
	err := db.Get(&user, q, username)
	return user, err
}

func recipeLabelExists(recipeID int, labelID int) (bool, error) {
	var exists []bool
	q := "SELECT count(*) FROM recipe_label WHERE recipe_id = ? and label_id = ?"
	connect()
	err := db.Select(&exists, q, recipeID, labelID)
	return exists[0], err
}

// Create //
func createLabel(labelName string) (Label, error) {
	q := "INSERT INTO label (label) VALUES (?)"
	connect()
	_, err := db.Exec(q, labelName)
	if err != nil {
		return Label{}, err
	}
	label, err := labelByName(labelName)
	if err != nil {
		return Label{}, err
	}
	fmt.Printf("created new label %s(%d)\n", label.Label, label.ID)
	return label, nil
}

func createRecipe(title string, body string, activeTime int, totalTime int) (Recipe, error) {
	q := "INSERT INTO recipe (title, recipe_body, active_time, total_time) VALUES (?, ?, ?, ?)"
	connect()
	result, err := db.Exec(q, title, body, activeTime, totalTime)
	if err != nil {
		return Recipe{}, err
	}
	recipeID, err := result.LastInsertId()
	if err != nil {
		return Recipe{}, err
	}
	return recipeByID(int(recipeID), false)
}

func createRecipeLabel(recipeID int, labelID int) error {
	q := "INSERT INTO recipe_label (recipe_id, label_id) VALUES (?, ?)"
	connect()
	_, err := db.Exec(q, recipeID, labelID)
	if err == nil {
		fmt.Printf("linked recipe %d to label %d\n", recipeID, labelID)
	}
	return err
}

func createNote(recipeID int, note string) (Note, error) {
	epoch := time.Now().Unix()
	q := "INSERT INTO note (recipe_id, note, create_date) VALUES (?, ?, ?)"
	connect()
	result, err := db.Exec(q, recipeID, note, epoch)
	if err != nil {
		return Note{}, err
	}
	noteID, err := result.LastInsertId()
	if err != nil {
		return Note{}, err
	}
	return getNoteByID(int(noteID))
}

// Edit //
func updateRecipe(recipeId int, title string, body string, activeTime int, totalTime int) error {
	q := `UPDATE recipe SET
		title = ?,
		recipe_body = ?,
		active_time = ?,
		total_time = ?
		WHERE recipe_id = ?`
	connect()
	_, err := db.Exec(q, title, body, activeTime, totalTime, recipeId)
	return err
}

func setNoteFlag(noteID int, flag bool) error {
	q := "UPDATE note SET flagged = ? WHERE note_id = ?"
	connect()
	_, err := db.Exec(q, flag, noteID)
	return err
}

func setNoteText(noteID int, text string) error {
	q := "UPDATE note SET note = ? WHERE note_id = ?"
	connect()
	_, err := db.Exec(q, text, noteID)
	return err
}

func softDeleteRecipe(recipeId int) error {
	q := "UPDATE recipe SET deleted = 1 WHERE recipe_id = ?"
	connect()
	_, err := db.Exec(q, recipeId)
	return err
}

func unDeleteRecipe(recipeId int) error {
	q := "UPDATE recipe SET deleted = 0 WHERE recipe_id = ?"
	connect()
	_, err := db.Exec(q, recipeId)
	return err
}

// Delete //
func deleteNote(noteID int) error {
	q := "DELETE FROM note WHERE note_id = ?"
	connect()
	_, err := db.Exec(q, noteID)
	if err == nil {
		fmt.Printf("deleted note %d\n", noteID)
	}
	return err
}

func deleteRecipeLabel(recipeID int, labelID int) error {
	q := "DELETE FROM recipe_label WHERE recipe_id = ? AND label_id = ?"
	connect()
	_, err := db.Exec(q, recipeID, labelID)
	if err == nil {
		fmt.Printf("unlinked label %d from recipe %d\n", labelID, recipeID)
	}
	return err
}

// MISC //
func connect() {
	if db != nil {
		return
	}
	db = sqlx.MustConnect(conf.DbDialect, conf.DbDSN)
}

/***********
 * METHODS *
 ***********/
func (u User) CheckPassword(cleartext string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(cleartext))
}

func (r Recipe) String() string {
	if r.ActiveTime != 0 && r.Time != 0 {
		return fmt.Sprintf("%s (%d min -- %d min active)", r.Title, r.Time, r.ActiveTime)
	}
	return fmt.Sprintf("%s", r.Title)
}

func (l Label) String() string {
	return fmt.Sprintf("%s", l.Label)
}
