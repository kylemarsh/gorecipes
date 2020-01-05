package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type configuration struct {
	Debug     bool
	Dev       bool
	DbDialect string
	DbConnStr string
}

var conf configuration

func main() {
	init_app()

	router := mux.NewRouter().StrictSlash(true)
	//router.HandleFunc("/", home)
	router.HandleFunc("/recipes/", getAllRecipes).Methods("GET")
	router.HandleFunc("/recipes/{id}", getRecipeByID).Methods("GET")
	router.HandleFunc("/recipes/{id}", deleteRecipe).Methods("DELETE")
	router.HandleFunc("/labels/", getAllLabels).Methods("GET")
	//router.HandleFunc("/recipes/{id}/labels", getLabelsForRecipe).Methods("GET")
	//router.HandleFunc("/labels/{id}/recipes", getRecipesForLabel).Methods("GET")
	handler := cors.Default().Handler(router)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func readConfiguration(c *configuration, configFilename string) error {
	file, err := os.Open(configFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&c)
}

func init_app() {
	configFilename := flag.String("config", "dev.config", "config file to use")
	doBootstrap := flag.Bool("bootstrap", false, "bootstrap db  with tables and sample data")
	force := flag.Bool("force", false, "force bootstrapping even if DB already exists")
	debug := flag.Bool("debug", false, "produce debugging output")
	flag.Parse()

	if err := readConfiguration(&conf, *configFilename); err != nil {
		panic(fmt.Sprintf("Error reading config: %v", err))
	}

	conf.Debug = *debug

	if conf.Debug {
		fmt.Println("Loaded config:")
		fmt.Println(conf)
	}

	connect()
	if *doBootstrap {
		bootstrap(*force)
	}
}
