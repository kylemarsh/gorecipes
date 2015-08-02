package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"os"
)

var force = flag.Bool("force", false, "drop and reinitialize the DB even when it already exists")
var conn *sql.DB

func main() {
	flag.Parse()
	conn, _ = sql.Open("mysql", "root@/gotest")
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
			"filename": dir + "labels.csv",
			"drop":     "DROP TABLE label",
			"create":   "CREATE TABLE `label` ( `label_id` int(11) NOT NULL auto_increment, `label` varchar(255) NOT NULL, PRIMARY KEY  (`label_id`), KEY `label` (`label`))",
			"insert":   "INSERT INTO label (label_id, label) VALUES (?, ?)",
		},
		"recipe": map[string]string{
			"filename": dir + "recipes.csv",
			"drop":     "DROP TABLE recipe",
			"create":   "CREATE TABLE `recipe` ( `recipe_id` int(11) NOT NULL auto_increment, `title` varchar(255) NOT NULL, `recipe_body` text NOT NULL, `total_time` int(11) NOT NULL, `active_time` int(11)   NOT NULL, PRIMARY KEY  (`recipe_id`), KEY `title` (`title`))",
			"insert":   "INSERT INTO recipe (recipe_id, title, recipe_body) VALUES (?, ?, ?)",
		},
		"recipe_label": map[string]string{
			"filename": dir + "recipe-label.csv",
			"drop":     "DROP TABLE recipe_label",
			"create":   "CREATE TABLE `recipe_label` ( `recipe_id` bigint(20) NOT NULL, `label_id` int(11) NOT NULL, PRIMARY KEY  (`recipe_id`,`label_id`))",
			"insert":   "INSERT INTO recipe_label (recipe_id, label_id) VALUES (?, ?)",
		},
	}

	tx, err := conn.Begin()
	if err != nil {
		fmt.Println("error creating transaction?", err)
	}

	initializeTable(tx, info["label"])
	initializeTable(tx, info["recipe"])
	initializeTable(tx, info["recipe_label"])
	tx.Commit()
}

func initializeTable(tx *sql.Tx, info map[string]string) {
	tx.Exec(info["drop"])
	tx.Exec(info["create"])

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
