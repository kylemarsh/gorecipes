package main

import (
	"encoding/json"
	"net/http"
	"strconv"

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
