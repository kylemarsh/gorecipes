package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
)

func getRecipeList(w http.ResponseWriter, r *http.Request) {
	recipes, err := allRecipes(false)

	if err != nil {
		apiError(w, http.StatusInternalServerError, "Problem loading recipes", err)
		return
	}
	json.NewEncoder(w).Encode(recipes)
}

func getAllLabels(w http.ResponseWriter, r *http.Request) {
	var labels []Label
	q := "SELECT * FROM label"
	connect()
	err := db.Select(&labels, q)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "Problem loading labels", err)
		return
	}
	json.NewEncoder(w).Encode(labels)
}

func getLabelsForRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID, _ := strconv.Atoi(mux.Vars(r)["id"])
	labels, err := labelsByRecipeID(recipeID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "Problem retrieving labels for recipe", err)
	}
	json.NewEncoder(w).Encode(labels)
}

func getRecipesForLabel(w http.ResponseWriter, r *http.Request) {
}

func login(w http.ResponseWriter, r *http.Request) {
	// TODO
	// get username/pass from request
	// compare username/pass against DB
	// 1 month expiration. TODO Decide on final scheme?
	username := r.FormValue("username")
	password := r.FormValue("password")
	fmt.Println("Username:")
	fmt.Println(username)

	fmt.Println("Password:")
	fmt.Println(password)

	claims := &jwt.StandardClaims{ExpiresAt: time.Now().Add(time.Hour * 24 * 30).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(conf.JwtSecret))

	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not sign token", err)
		return
	}
	fmt.Println("Token:")
	fmt.Println(token)
	fmt.Println(tokenStr)

	json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenStr})
}
