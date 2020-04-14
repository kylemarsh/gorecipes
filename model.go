package main

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var db *sqlx.DB

/*Recipe - basic unit of the recipe database */
type Recipe struct {
	ID         int `db:"recipe_id"`
	Title      string
	Body       string `db:"recipe_body"`
	Time       int    `db:"total_time"`
	ActiveTime int    `db:"active_time"`
	Labels     []Label
}

func (r Recipe) String() string {
	if r.ActiveTime != 0 && r.Time != 0 {
		return fmt.Sprintf("%s (%d min -- %d min active)", r.Title, r.Time, r.ActiveTime)
	}
	return fmt.Sprintf("%s", r.Title)
}

/*Label - a taxonomic tab for recipes */
type Label struct {
	ID    int `db:"label_id"`
	Label string
}

func (l Label) String() string {
	return fmt.Sprintf("%s", l.Label)
}

/* Functions */
func allRecipes(includeBody bool) ([]Recipe, error) {
	var recipes []Recipe
	var q string
	if includeBody {
		q = "SELECT * FROM recipe"
	} else {
		q = "SELECT recipe_id, title, total_time, active_time FROM recipe"
	}
	//TODO: add labels
	connect()
	err := db.Select(&recipes, q)
	return recipes, err
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

func connect() {
	if db != nil {
		return
	}
	db = sqlx.MustConnect(conf.DbDialect, conf.DbConnStr)
}
