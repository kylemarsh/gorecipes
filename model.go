package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

/* Recipe */
type Recipe struct {
	id         int
	Title      string
	Body       string
	Time       int
	ActiveTime int
}

func (r Recipe) Save() error {
	panic("notimplemented")
}

func (r Recipe) Labels() []Label {
	panic("notimplememnted")
}

func (r Recipe) String() string {
	if r.ActiveTime != 0 && r.Time != 0 {
		return fmt.Sprintf("%s (%d min -- %d min active)", r.Title, r.Time, r.ActiveTime)
	} else {
		return fmt.Sprintf("%s", r.Title)
	}
}

/* Label */
type Label struct {
	id    int
	Label string
}

func (l Label) String() string {
	return fmt.Sprintf("%s", l.Label)
}

/* Functions */
func recipeById(id int) Recipe {
	var recipe Recipe
	q := "SELECT recipe_id, title, recipe_body, total_time, active_time FROM recipe WHERE recipe_id = ?"
	params := []interface{}{id}
	dest := []interface{}{&recipe.id, &recipe.Title, &recipe.Body, &recipe.Time, &recipe.ActiveTime}
	dbGetQuery(q, params, dest)

	//db := connect()
	//row := db.QueryRow(q, id)
	//row.Scan(&recipe.id, &recipe.Title, &recipe.Body, &recipe.Time, &recipe.ActiveTime)
	return recipe
}

func labelById(id int) Label {
	var label Label
	q := "SELECT label_id, label FROM label WHERE label_id = ?"

	db := connect()
	row := db.QueryRow(q, id)
	row.Scan(&label.id, &label.Label)
	return label
}

func dbGetQuery(q string, params []interface{}, dest []interface{}) {
	db := connect()
	row := db.QueryRow(q, params...)
	row.Scan(dest...)
}

func connect() *sql.DB {
	//TODO: Put this in a config file!
	db, err := sql.Open("mysql", "root@/gotest")
	if err != nil {
		panic(err.Error())

	}
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	return db
}
