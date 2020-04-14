package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type configuration struct {
	Debug     bool
	Dev       bool
	DbDialect string
	DbConnStr string
	// TODO
	// username/password
	// JWT Signing secret
}

var conf configuration

func main() {
	initApp()

	// TODO: put privileged routes under subrouter with authRequired middleware
	router := mux.NewRouter().StrictSlash(true)
	//router.HandleFunc("/", home)
	router.HandleFunc("/recipes/", getAllRecipes).Methods("GET")
	router.HandleFunc("/recipelist/", getRecipeList).Methods("GET")
	router.HandleFunc("/recipes/{id}", getRecipeByID).Methods("GET")
	router.HandleFunc("/recipes/{id}", deleteRecipe).Methods("DELETE")
	router.HandleFunc("/labels/", getAllLabels).Methods("GET")
	//router.HandleFunc("/recipes/{id}/labels", getLabelsForRecipe).Methods("GET")
	//router.HandleFunc("/labels/{id}/recipes", getRecipesForLabel).Methods("GET")

	debugRouter := router.PathPrefix("/debug").Subrouter()
	debugRouter.Use(debugRequired)
	debugRouter.HandleFunc("/getToken/", jwtGenerate).Methods("GET")
	debugRouter.HandleFunc("/checkToken/", jwtValidate).Methods("GET")

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

func initApp() {
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

func validJwt(r *http.Request) bool {
	var header = r.Header.Get("x-access-token")
	tokenString := strings.TrimSpace(header)
	if tokenString == "" {
		return false
	}

	tk, err := jwt.Parse(header, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil // FIXME get secret from config?
	})

	if err != nil || tk.Valid != true {
		return false
	}

	return true
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

func jwtValidate(w http.ResponseWriter, r *http.Request) {

	var header = r.Header.Get("x-access-token")
	tokenString := strings.TrimSpace(header)
	if tokenString == "" {
		apiError(w, http.StatusUnauthorized, "missing auth token:", nil)
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})

	var ErrTokenExpired = errors.New("Token is expired")
	if err != nil {
		if err == ErrTokenExpired {
			apiError(w, http.StatusUnauthorized, "", err)
		} else {
			apiError(w, http.StatusBadRequest, "could not parse token:", err)
		}
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"parsedToken": token})
}

func jwtGenerate(w http.ResponseWriter, r *http.Request) {

	// 1 month expiration. TODO Decide on final scheme?
	claims := &jwt.StandardClaims{ExpiresAt: time.Now().Add(time.Hour * 24 * 30).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("secret")) // FIXME get secret from config

	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not sign token", err)
		return
	}
	fmt.Println("Token:")
	fmt.Println(token)
	fmt.Println(tokenStr)

	json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenStr})
}

func apiError(w http.ResponseWriter, statusCode int, msg string, err error) {
	w.WriteHeader(statusCode)
	if conf.Debug {
		fmt.Fprintln(w, msg, err)
	} else {
		fmt.Fprintln(w, msg)
	}
}
