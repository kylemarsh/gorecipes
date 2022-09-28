package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type configuration struct {
	Debug     bool
	DbDialect string
	DbDSN     string
	JwtSecret string
	Origins   []string
}

type appError struct {
	Code    int
	Message string
	Error   error
}

type wrappedHandler func(w http.ResponseWriter, r *http.Request) *appError

var conf configuration

func main() {
	initApp()

	router := mux.NewRouter().StrictSlash(true)
	router.Handle("/login/", wrappedHandler(login)).Methods("POST")

	router.Handle("/recipes/", wrappedHandler(getRecipeList)).Methods("GET")
	router.Handle("/labels/", wrappedHandler(getAllLabels)).Methods("GET")
	router.Handle("/recipe/{id}/labels/", wrappedHandler(getLabelsForRecipe)).Methods("GET")
	//router.Handle("/labels/{id}/recipes", wrappedHandler(getRecipesForLabel)).Methods("GET")

	privRouter := router.PathPrefix("/priv").Subrouter()
	privRouter.Use(authRequired)
	privRouter.Handle("/recipes/", wrappedHandler(getAllRecipes)).Methods("GET")
	privRouter.Handle("/recipe/{id}/", wrappedHandler(getRecipeByID)).Methods("GET")
	privRouter.Handle("/recipe/{id}/", wrappedHandler(deleteRecipeSoft)).Methods("DELETE")
	privRouter.Handle("/recipe/{id}/hard", wrappedHandler(deleteRecipeHard)).Methods("DELETE")
	privRouter.Handle("/recipe/{id}/restore", wrappedHandler(recipeRestore)).Methods("PUT")
	privRouter.Handle("/recipe/{id}", wrappedHandler(updateExistingRecipe)).Methods("PUT")
	privRouter.Handle("/recipe/", wrappedHandler(createNewRecipe)).Methods("POST")
	privRouter.Handle("/recipe/{recipe_id}/label/{label_id}", wrappedHandler(tagRecipe)).Methods("PUT")
	privRouter.Handle("/recipe/{recipe_id}/label/{label_id}", wrappedHandler(untagRecipe)).Methods("DELETE")
	privRouter.Handle("/label/{label_name}", wrappedHandler(addLabel)).Methods("PUT")
	privRouter.Handle("/recipe/{id}/notes/", wrappedHandler(getNotesForRecipe)).Methods("GET")
	privRouter.Handle("/recipe/{id}/note/", wrappedHandler(createNoteOnRecipe)).Methods("POST")
	privRouter.Handle("/note/{id}", wrappedHandler(removeNote)).Methods("DELETE")
	privRouter.Handle("/note/{id}", wrappedHandler(editNote)).Methods("PUT")
	privRouter.Handle("/note/{id}/flag", wrappedHandler(flagNote)).Methods("PUT")
	privRouter.Handle("/note/{id}/unflag", wrappedHandler(unFlagNote)).Methods("PUT")

	debugRouter := router.PathPrefix("/debug").Subrouter()
	debugRouter.Use(debugRequired)
	debugRouter.Handle("/getToken/", wrappedHandler(getJwt)).Methods("GET")
	debugRouter.Handle("/checkToken/", wrappedHandler(validateJwt)).Methods("GET")
	debugRouter.Handle("/hashPassword/", wrappedHandler(getHash)).Methods("POST")

	var corsOptions cors.Options
	if conf.Debug {
		corsOptions = cors.Options{
			AllowedHeaders: []string{"*"},
			AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			Debug:          true,
		}
	} else {
		corsOptions = cors.Options{
			AllowedHeaders: []string{"x-access-token"},
			AllowedOrigins: conf.Origins,
			AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		}
	}
	handler := cors.New(corsOptions).Handler(router)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func initApp() {
	configFilename := flag.String("config", "gorecipes.conf", "config file to use")
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

	if conf.JwtSecret == "" {
		panic("JWT Secret is a required config")
	}

	if !conf.Debug && len(conf.Origins) == 0 {
		panic("You must provide allowed origins for CORS when not running under debug")
	}

	connect()
	if *doBootstrap {
		bootstrap(*force)
	}
}

func (fn wrappedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil { // Note this is specifically our *appError
		http.Error(w, err.Message, err.Code)
		fmt.Printf("%v\n", err.Error)
		return
	}
}
