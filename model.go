package main

import (
	"fmt"

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
func allRecipes(includeBody bool) ([]Recipe, error) {
	var recipes []Recipe
	var q string
	if includeBody {
		q = "SELECT * FROM recipe"
		// TODO can we populate the labels and recipes at the same time?
		//q = "SELECT recipe.*, label.* FROM recipe join recipe_label using(recipe_id) join label using(label_id)"
	} else {
		q = "SELECT recipe_id, title, total_time, active_time FROM recipe"
	}
	connect()
	err := db.Select(&recipes, q)
	if err != nil {
		return recipes, err
	}

	var savedErr error
	//TODO load in labels for each recipe
	for _, recipe := range recipes {
		labels, err := labelsByRecipeID(recipe.ID)
		if err != nil {
			savedErr = err
			fmt.Println("error loading labels for recipe", recipe.ID, err)
		}
		recipe.Labels = labels
		fmt.Println()
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

func labelByID(id int) Label {
	var label Label
	q := "SELECT * FROM label WHERE label_id = ?"

	connect()
	if err := db.Get(&label, q, id); err != nil {
		fmt.Println("Error finding label: ", err)
	}
	return label
}

func labelsByRecipeID(id int) ([]Label, error) {
	var labels []Label
	q := "SELECT label.* FROM label join recipe_label using(label_id) WHERE recipe_id = ?"

	connect()
	err := db.Select(&labels, q, id)
	return labels, err
}

func notesByRecipeID(recipe_id int, flagged_only bool) ([]Note, error) {
	var notes []Note
	var q string
	if flagged_only {
		q = "SELECT * FROM note WHERE recipe_id = ? and flagged = 1"
	} else {
		q = "SELECT * FROM note WHERE recipe_id = ?"
	}

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
