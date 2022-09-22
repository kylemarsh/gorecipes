package main

import (
	"testing"
)

func TestConnect(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}
	if db != nil {
		db.Close()
		db = nil
	}

	connect()
	if db == nil {
		t.Errorf("connect() did not create DB")
	}
}

func TestBootstrap(t *testing.T) {
	conf = configuration{
		Debug:     false,
		DbDialect: "sqlite3",
		DbDSN:     ":memory:",
		JwtSecret: "secret",
	}

	if db != nil {
		db.Close()
		db = nil
	}
	connect()

	checkDb(t, 0, 0, 0)
	bootstrap(false)
	checkDb(t, 37, 13, 34)

	db.Exec("insert into label values (40, 'florp')")
	bootstrap(false)
	checkDb(t, 38, 13, 34)
	bootstrap(true)
	checkDb(t, 37, 13, 34)
}

func checkDb(t *testing.T, expectedLabels int, expectedRecipes int, expectedRecipeLabels int) {
	var (
		numLabels       int
		numRecipes      int
		numRecipeLabels int
	)

	db.QueryRow("select count(*) from label").Scan(&numLabels)
	db.QueryRow("select count(*) from recipe").Scan(&numRecipes)
	db.QueryRow("select count(*) from recipe_label").Scan(&numRecipeLabels)
	if numLabels != expectedLabels {
		t.Errorf("Got %v labels, expected %v", numLabels, expectedLabels)
	}
	if numRecipes != expectedRecipes {
		t.Errorf("Got %v recipes, expected %v", numRecipes, expectedRecipes)
	}
	if numRecipeLabels != expectedRecipeLabels {
		t.Errorf("Got %v recipe-labels, expected %v", numRecipeLabels, expectedRecipeLabels)
	}
}

/****
// This assumes a DB is available and it's populated with the bootstrapping data
func TestLoadRecipe(t *testing.T) {
	t.Log("Testing recipeById")
	expectedBody := `2 cups flour
1 T sugar
1 t salt
½ cup shortening
1 cup sour cream
1 egg yolk

¾ lb ground beef
1 large onion, finely chopped
¼ cup finely chopped fresh mushrooms
½ cup sour cream
½ t salt
½ t oregano
¼ t pepper

1 egg
2 t water

Combine flour, sugar, salt. Cut in shortening until crumbly
Stir in sour cream and egg yolk until just moistened
Shape into ball, cover and refrigerate for 2 hours
In a large skillet over medium heat cook beef, onion, and mushrooms
Drain, stir in sour cream, salt, oregano, pepper
Roll out dough to 1/8” thickness
Cut into 3” disks
Place rounded teaspoon of filling on one side of each circle
Fold dough over filling, press edges with a fork to seal
Prick tops with a fork
Place on greased baking sheets
Beat eggs with water, brush over turnovers
Bake at 450 for 12-15 minutes or until lightly browned`

	expected := Recipe{
		id:         1,
		Title:      "Beef Turnover",
		Body:       expected_body,
		Time:       30,
		ActiveTime: 10,
	}
	id := 1
	recipe := recipeById(id)
	if recipe.id != expected.id {
		t.Errorf("id does not match (got %d: expected %d)", recipe.id, expected.id)
	}
	if recipe.Title != expected.Title {
		t.Errorf("Title does not match (got %d: expected %d)", recipe.Title, expected.Title)
	}
	if recipe.Body != expected.Body {
		t.Errorf("Body does not match (got %d: expected %d)", recipe.Body, expected.Body)
	}
	if recipe.Time != expected.Time {
		t.Errorf("Time does not match (got %d: expected %d)", recipe.Time, expected.Time)
	}
	if recipe.ActiveTime != expected.ActiveTime {
		t.Errorf("ActiveTime does not match (got %d: expected %d)", recipe.ActiveTime, expected.ActiveTime)
	}
}

func TestRecipeString(t *testing.T) {
	t.Log("Testing Recipe String()")

	r := Recipe{
		id:    123,
		Title: "Test Recipe",
		Body:  "instructions for the making",
	}
	expected := "Test Recipe"

	if r.String() != expected {
		t.Errorf("no times:\ngot:      %s\nexpected: %s", r.String(), expected)
	}

	r.ActiveTime = 10
	if r.String() != expected {
		t.Errorf("active only:\ngot:      %s\nexpected: %s", r.String(), expected)
	}

	r.ActiveTime = 0
	r.Time = 30
	if r.String() != expected {
		t.Errorf("time only:\ngot:      %s\nexpected: %s", r.String(), expected)
	}

	r.ActiveTime = 10
	expected = "Test Recipe (30 min -- 10 min active)"
	if r.String() != expected {
		t.Errorf("Including Times:\ngot:      %s\nexpected: %s", r.String(), expected)
	}
}

func TestLoadLabel(t *testing.T) {
	t.Log("Testing labelById")
	expected := Label{
		id:    1,
		Label: "chicken",
	}

	id := 1
	label := labelById(id)
	if label.id != expected.id {
		t.Errorf("id does not match (got %d: expected %d)", label.id, expected.id)
	}
	if label.Label != expected.Label {
		t.Errorf("Label does not match (got %d: expected %d)", label.id, expected.id)
	}
}
****/
