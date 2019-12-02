package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

var force = flag.Bool("force", false, "drop and reinitialize the DB even when it already exists")
var conn *sql.DB

func main() {
	flag.Parse()
	//conn, _ = sql.Open("mysql", "root@/gotest")
	conn, _ = sql.Open("sqlite3", "recipes_sqlite.db")
	defer conn.Close()

	r := conn.QueryRow("select count(*) from label")
	var numLabels int
	r.Scan(&numLabels)
	if numLabels == 0 || *force {
		bootstrap()
	} else {
		fmt.Println("The database seems to be populated already...if you really want to re-initialize it use --force")
	}
}

func bootstrap() {
	fmt.Println("bootstrapping")
	dir := "/Users/kylem/projects/gorecipes/bootstrapping/"

	var info = map[string]map[string]string{
		"label": map[string]string{
			"filename":     dir + "labels.csv",
			"drop":         "DROP TABLE IF EXISTS label",
			"createMySQL":  "CREATE TABLE `label` ( `label_id` int(11) NOT NULL auto_increment, `label` varchar(255) NOT NULL, PRIMARY KEY  (`label_id`), KEY `label` (`label`))",
			"createSQLITE": "CREATE TABLE `label` ( `label_id` int(11) NOT NULL PRIMARY KEY, `label` varchar(255) NOT NULL)",
			"insert":       "INSERT INTO label (label_id, label) VALUES (?, ?)",
		},
		"recipe": map[string]string{
			"filename":     dir + "recipes.csv",
			"drop":         "DROP TABLE IF EXISTS recipe",
			"createMySQL":  "CREATE TABLE `recipe` ( `recipe_id` int(11) NOT NULL auto_increment, `title` varchar(255) NOT NULL, `recipe_body` text NOT NULL, `total_time` int(11) NOT NULL, `active_time` int(11)   NOT NULL, PRIMARY KEY  (`recipe_id`), KEY `title` (`title`))",
			"createSQLITE": "CREATE TABLE `recipe` ( `recipe_id` int(11) NOT NULL PRIMARY KEY, `title` varchar(255) NOT NULL, `recipe_body` text NOT NULL, `total_time` int(11) NOT NULL, `active_time` int(11)   NOT NULL)",
			"insert":       "INSERT INTO recipe (recipe_id, title, recipe_body) VALUES (?, ?, ?)",
		},
		"recipe_label": map[string]string{
			"filename":     dir + "recipe-label.csv",
			"drop":         "DROP TABLE IF EXISTS recipe_label",
			"createMySQL":  "CREATE TABLE `recipe_label` ( `recipe_id` bigint(20) NOT NULL, `label_id` int(11) NOT NULL, PRIMARY KEY  (`recipe_id`,`label_id`))",
			"createSQLITE": "CREATE TABLE `recipe_label` ( `recipe_id` bigint(20) NOT NULL, `label_id` int(11) NOT NULL, PRIMARY KEY  (`recipe_id`,`label_id`))",
			"insert":       "INSERT INTO recipe_label (recipe_id, label_id) VALUES (?, ?)",
		},
	}

	tx, err := conn.Begin()
	if err != nil {
		fmt.Println("error creating transaction?", err)
	}

	fmt.Println("Initializing Labels")
	initializeTable(tx, info["label"])

	fmt.Println("Initializing Recipes")
	initializeTable(tx, info["recipe"])

	fmt.Println("Initializing Recipe-Label")
	initializeTable(tx, info["recipe_label"])

	tx.Commit()
}

func initializeTable(tx *sql.Tx, info map[string]string) {
	if _, err := tx.Exec(info["drop"]); err != nil {
		fmt.Println("Error dropping: ", err)
	}
	//FIXME config
	if _, err := tx.Exec(info["createSQLITE"]); err != nil {
		fmt.Println("Error creating: ", err)
	}

	file, err := os.Open(info["filename"])
	if err != nil {
		fmt.Println("ugh:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	fmt.Println(info["insert"])
	for {
		record, err := reader.Read()
		if err == io.EOF {
			fmt.Println("done")
			break
		} else if err != nil {
			fmt.Println("err:", err)
			break
		}

		id := record[0]
		if id == "label_id" || id == "recipe_id" {
			fmt.Println(record)
			continue //skip headers
		}

		args := make([]interface{}, len(record))
		for i, v := range record {
			args[i] = v
		}
		tx.Exec(info["insert"], args...)
	}
}
