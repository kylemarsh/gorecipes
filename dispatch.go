package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func getAllRecipes(w http.ResponseWriter, r *http.Request) {
	var recipes []Recipe
	q := "SELECT * FROM recipe"
	connect()
	err := db.Select(&recipes, q)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "Problem loading recipes", err)
		return
	}
	json.NewEncoder(w).Encode(recipes)
}

func getRecipeByID(w http.ResponseWriter, r *http.Request) {
	recipeID, _ := strconv.Atoi(mux.Vars(r)["id"])

	if recipe, err := recipeByID(recipeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "No recipe with id=%v exists", recipeID)
		} else {
			apiError(w, http.StatusInternalServerError, "Problem loading recipe", err)
		}
	} else {
		json.NewEncoder(w).Encode(recipe)
	}
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

//TODO: Wrap in "recipeRequired" and "loginRequired" functions that handle boilerplate?
func deleteRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID, _ := strconv.Atoi(mux.Vars(r)["id"])

	if _, err := recipeByID(recipeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "No recipe with id=%v exists", recipeID)
		} else {
			apiError(w, http.StatusInternalServerError, "Problem loading recipe", err)
		}
		return
	}

	connect()
	q := "DELETE FROM recipe WHERE recipe_id = ?"
	if _, err := db.Exec(q, recipeID); err != nil {
		apiError(w, http.StatusInternalServerError, "Problem deleting recipe", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func getLabelsForRecipe(w http.ResponseWriter, r *http.Request) {
}

func getRecipesForLabel(w http.ResponseWriter, r *http.Request) {
}

func apiError(w http.ResponseWriter, statusCode int, msg string, err error) {
	w.WriteHeader(statusCode)
	if conf.Debug {
		fmt.Fprintln(w, msg, err)
	} else {
		fmt.Fprintln(w, msg)
	}
}
