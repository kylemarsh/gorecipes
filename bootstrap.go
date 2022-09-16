package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

func bootstrap(force bool) {
	if populated() && !force {
		fmt.Println("The database seems to be populated already...if you really want to re-initialize it use --force")
		return
	}

	fmt.Println("bootstrapping")
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dir := cwd + "/bootstrapping/" //This won't work if we put the binary somewhere other than the root of the project

	var info = map[string]map[string]string{
		"label": {
			"filename":       dir + "labels.csv",
			"drop":           "DROP TABLE IF EXISTS label",
			"create_mysql":   "CREATE TABLE `label` ( `label_id` int(11) NOT NULL auto_increment, `label` varchar(255) NOT NULL, PRIMARY KEY  (`label_id`), KEY `label` (`label`))",
			"create_sqlite3": "CREATE TABLE `label` ( `label_id` INTEGER PRIMARY KEY, `label` varchar(255) NOT NULL)",
			"insert":         "INSERT INTO label (label_id, label) VALUES (?, ?)",
		},
		"recipe": {
			"filename":       dir + "recipes.csv",
			"drop":           "DROP TABLE IF EXISTS recipe",
			"create_mysql":   "CREATE TABLE `recipe` ( `recipe_id` int(11) NOT NULL auto_increment, `title` varchar(255) NOT NULL, `recipe_body` text NOT NULL, `total_time` int(11) NOT NULL, `active_time` int(11)   NOT NULL, PRIMARY KEY  (`recipe_id`), KEY `title` (`title`))",
			"create_sqlite3": "CREATE TABLE `recipe` ( `recipe_id` INTEGER PRIMARY KEY, `title` varchar(255) NOT NULL, `recipe_body` text NOT NULL, `total_time` int NOT NULL, `active_time` int   NOT NULL)",
			"insert":         "INSERT INTO recipe (recipe_id, title, recipe_body, total_time, active_time) VALUES (?, ?, ?, ?, ?)",
		},
		"recipe_label": {
			"filename":       dir + "recipe-label.csv",
			"drop":           "DROP TABLE IF EXISTS recipe_label",
			"create_mysql":   "CREATE TABLE `recipe_label` ( `recipe_id` bigint(20) NOT NULL, `label_id` int(11) NOT NULL, PRIMARY KEY  (`recipe_id`,`label_id`))",
			"create_sqlite3": "CREATE TABLE `recipe_label` ( `recipe_id` bigint NOT NULL, `label_id` int NOT NULL, PRIMARY KEY  (`recipe_id`,`label_id`))",
			"insert":         "INSERT INTO recipe_label (recipe_id, label_id) VALUES (?, ?)",
		},
		"note": {
			"filename":       dir + "notes.csv",
			"drop":           "DROP TABLE IF EXISTS note",
			"create_mysql":   "CREATE TABLE `note` ( `note_id` bigint(20) NOT NULL AUTO_INCREMENT, `recipe_id` bigint(20) NOT NULL, `create_date` bigint(20) NOT NULL, `note` TEXT NOT NULL, `flagged` BOOLEAN NOT NULL DEFAULT 1, PRIMARY KEY (`note_id`), KEY `recipe` (`recipe_id`))",
			"create_sqlite3": "CREATE TABLE `note` ( `note_id` INTEGER PRIMARY KEY, `recipe_id` INTEGER NOT NULL, `create_date` TEXT NOT NULL, `note` TEXT NOT NULL, `flagged` BOOLEAN DEFAULT TRUE)",
			"insert":         "INSERT INTO note (note_id, recipe_id, create_date, note, flagged) VALUES (?, ?, ?, ?, ?)",
		},
		"user": {
			"filename":       dir + "users.csv",
			"drop":           "DROP TABLE IF EXISTS user",
			"create_mysql":   "CREATE TABLE `user` ( `user_id` bigint(20) NOT NULL AUTO_INCREMENT, `username` varchar(63) NOT NULL, `password` varchar(255), `plaintext_pw_bootstrapping_only` varchar(255) NOT NULL, PRIMARY KEY (`user_id`), KEY `username` (`username`))",
			"create_sqlite3": "CREATE TABLE `user` ( `user_id` INTEGER PRIMARY KEY, `username` varchar(63) NOT NULL, `password` varchar(255), `plaintext_pw_bootstrapping_only` varchar(255) NOT NULL)",
			"insert":         "INSERT INTO user (user_id, username, password, plaintext_pw_bootstrapping_only) VALUES (?, ?, ?, ?)",
		},
	}

	tx, err := db.Begin()
	if err != nil {
		panic(fmt.Sprintf("error creating transaction? %v", err))
	}

	fmt.Println("Initializing Labels")
	initializeTable(tx, info["label"])

	fmt.Println("Initializing Recipes")
	initializeTable(tx, info["recipe"])

	fmt.Println("Initializing Recipe-Label")
	initializeTable(tx, info["recipe_label"])

	fmt.Println("Initializing Notes")
	initializeTable(tx, info["note"])

	fmt.Println("Initializing Users")
	initializeTable(tx, info["user"])

	tx.Commit()
}

func initializeTable(tx *sql.Tx, info map[string]string) {
	if _, err := tx.Exec(info["drop"]); err != nil {
		fmt.Println("Error dropping: ", err)
	}
	if _, err := tx.Exec(info["create_"+conf.DbDialect]); err != nil {
		fmt.Println("Error creating: ", err)
	}

	file, err := os.Open(info["filename"])
	if err != nil {
		fmt.Println("Error opening bootstrapping file:", err)
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

		if conf.Debug {
			fmt.Println(record)
		}

		id := record[0]
		if id == "label_id" || id == "recipe_id" || id == "note_id" {
			continue //skip headers
		}

		args := make([]interface{}, len(record))
		for i, v := range record {
			args[i] = v
		}
		if _, err := tx.Exec(info["insert"], args...); err != nil {
			fmt.Println("Error inserting row:", err)
		}
	}
}

func populated() bool {
	r := db.QueryRow("select count(*) from label")
	var numLabels int
	r.Scan(&numLabels)
	return numLabels != 0
}
