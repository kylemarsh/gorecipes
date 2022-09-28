package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func getRecipeList(w http.ResponseWriter, r *http.Request) *appError {
	recipes, err := activeRecipes(false)

	if err != nil {
		return &appError{http.StatusInternalServerError, "Problem loading recipes", err}
	}
	json.NewEncoder(w).Encode(recipes)
	return nil
}

func getAllLabels(w http.ResponseWriter, r *http.Request) *appError {
	var labels []Label
	q := "SELECT * FROM label"
	connect()
	err := db.Select(&labels, q)
	if err != nil {
		return &appError{http.StatusInternalServerError, "Problem loading labels", err}
	}
	json.NewEncoder(w).Encode(labels)
	return nil
}

func getLabelsForRecipe(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, _ := strconv.Atoi(mux.Vars(r)["id"])
	labels, err := labelsByRecipeID(recipeID)
	if err != nil {
		return &appError{http.StatusInternalServerError, "Problem retrieving labels for recipe", err}
	}
	json.NewEncoder(w).Encode(labels)
	return nil
}

func getRecipesForLabel(w http.ResponseWriter, r *http.Request) *appError {
	return &appError{http.StatusInternalServerError, "unimplemented", nil}
}

func login(w http.ResponseWriter, r *http.Request) *appError {
	// 1 month expiration. TODO Decide on final scheme?
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := userByName(username)
	if err != nil {
		return &appError{http.StatusForbidden, "login invalid", err}
	}
	err = user.CheckPassword(password)
	if err != nil {
		return &appError{http.StatusForbidden, "login invalid", err}
	}

	tokenStr, err := jwtGenerate()
	if err != nil {
		return &appError{http.StatusInternalServerError, "could not sign token", err}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenStr})
	return nil
}
