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
}

//func (r Recipe) Save() error {
//panic("notimplemented")
//}

//func (r Recipe) Labels() []Label {
//panic("notimplememnted")
//}

func (r Recipe) String() string {
	if r.ActiveTime != 0 && r.Time != 0 {
		return fmt.Sprintf("%s (%d min -- %d min active)", r.Title, r.Time, r.ActiveTime)
	}
	return fmt.Sprintf("%s", r.Title)
}

/*Label - a taxonomic tab for recipes */
type Label struct {
	id    int
	Label string
}

func (l Label) String() string {
	return fmt.Sprintf("%s", l.Label)
}

/* Functions */
func recipeByID(id int) (Recipe, error) {
	var recipe Recipe
	q := "SELECT * FROM recipe WHERE recipe_id = ?"

	connect()
	err := db.Get(&recipe, q, id)
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

func connect() {
	if db != nil {
		return
	}
	db = sqlx.MustConnect(conf.DbDialect, conf.DbConnStr)
}
