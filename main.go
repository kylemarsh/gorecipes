package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type configuration struct {
	Debug     bool
	DbDialect string
	DbDSN     string
	JwtSecret string
}

var conf configuration

func main() {
	initApp()

	router := mux.NewRouter().StrictSlash(true)
	//router.HandleFunc("/", home)
	router.HandleFunc("/login/", login).Methods("POST")

	router.HandleFunc("/recipes/", getRecipeList).Methods("GET")
	router.HandleFunc("/labels/", getAllLabels).Methods("GET")
	router.HandleFunc("/recipe/{id}/labels/", getLabelsForRecipe).Methods("GET")
	//router.HandleFunc("/labels/{id}/recipes", getRecipesForLabel).Methods("GET")

	privRouter := router.PathPrefix("/priv").Subrouter()
	privRouter.Use(authRequired)
	privRouter.HandleFunc("/recipes/", getAllRecipes).Methods("GET")
	privRouter.HandleFunc("/recipe/{id}/", getRecipeByID).Methods("GET")
	privRouter.HandleFunc("/recipe/{id}/", deleteRecipeSoft).Methods("DELETE")
	privRouter.HandleFunc("/recipe/{id}/hard", deleteRecipeHard).Methods("DELETE")
	privRouter.HandleFunc("/recipe/{id}/restore", recipeRestore).Methods("PUT")
	privRouter.HandleFunc("/recipe/{id}", updateExistingRecipe).Methods("PUT")
	privRouter.HandleFunc("/recipe/", createNewRecipe).Methods("POST")
	privRouter.HandleFunc("/recipe/{recipe_id}/label/{label_id}", tagRecipe).Methods("PUT")
	privRouter.HandleFunc("/recipe/{recipe_id}/label/{label_id}", untagRecipe).Methods("DELETE")
	privRouter.HandleFunc("/label/{label_name}", addLabel).Methods("PUT")
	privRouter.HandleFunc("/recipe/{id}/notes/", getNotesForRecipe).Methods("GET")
	privRouter.HandleFunc("/recipe/{id}/note/", createNoteOnRecipe).Methods("POST")
	privRouter.HandleFunc("/note/{id}", removeNote).Methods("DELETE")
	privRouter.HandleFunc("/note/{id}", editNote).Methods("PUT")
	privRouter.HandleFunc("/note/{id}/flag", flagNote).Methods("PUT")
	privRouter.HandleFunc("/note/{id}/unflag", unFlagNote).Methods("PUT")

	debugRouter := router.PathPrefix("/debug").Subrouter()
	debugRouter.Use(debugRequired)
	debugRouter.HandleFunc("/getToken/", getJwt).Methods("GET")
	debugRouter.HandleFunc("/checkToken/", validateJwt).Methods("GET")
	debugRouter.HandleFunc("/hashPassword/", getHash).Methods("POST")

	//handler := cors.AllowAll().Handler(router)
	//handler := cors.Default().Handler(router)
	var corsOptions cors.Options
	if conf.Debug {
		corsOptions = cors.Options{
			AllowedHeaders: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		}
	} else {
		corsOptions = cors.Options{
			//AllowedHeaders: []string{"*"}, //FIXME
			//AllowedOrigin:  []string{"api.recipelister.quixoticflame.net"},
			//AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
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

	connect()
	if *doBootstrap {
		bootstrap(*force)
	}
}

func authRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var header = r.Header.Get("x-access-token")
		tokenString := strings.TrimSpace(header)
		if tokenString == "" {
			apiError(w, http.StatusUnauthorized, "missing auth token", nil)
			return
		}

		_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(conf.JwtSecret), nil
		})

		if err != nil {
			errstring := err.Error()
			if errstring == "Token is expired" {
				apiError(w, http.StatusUnauthorized, "auth token expired; please log in again", err)
			} else {
				apiError(w, http.StatusBadRequest, "invalid auth token", err)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Debug Mode Middleware -- prohibits accessing certain routes when debug mode is disabled
func debugRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !conf.Debug {
			apiError(w, http.StatusForbidden, "token validation only available for debugging", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validateJwt(w http.ResponseWriter, r *http.Request) {

	var header = r.Header.Get("x-access-token")
	tokenString := strings.TrimSpace(header)
	err := jwtValidate(tokenString)
	if err != nil {
		apiError(w, http.StatusBadRequest, "invalid auth token", err)
	}
	w.WriteHeader(http.StatusOK)
}

func getHash(w http.ResponseWriter, r *http.Request) {

	var password = r.FormValue("password")
	hash, err := hashPassword(password)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "problem hashing password", err)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"hash": hash})
}

func getJwt(w http.ResponseWriter, r *http.Request) {

	tokenStr, err := jwtGenerate()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not sign token", err)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenStr})
}

func apiError(w http.ResponseWriter, statusCode int, msg string, err error) {
	// Log the error
	fmt.Println(msg, err)

	// Write the error to the response
	w.WriteHeader(statusCode)
	if conf.Debug {
		fmt.Fprintln(w, msg, err)
	} else {
		fmt.Fprintln(w, msg)
	}
}
