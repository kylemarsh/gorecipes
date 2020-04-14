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
	recipes, err := allRecipes(true)

	if err != nil {
		apiError(w, http.StatusInternalServerError, "Problem loading recipes", err)
		return
	}
	json.NewEncoder(w).Encode(recipes)
}

func getRecipeByID(w http.ResponseWriter, r *http.Request) {
	recipeID, _ := strconv.Atoi(mux.Vars(r)["id"])

	if recipe, err := recipeByID(recipeID, true); err != nil {
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

//TODO: Wrap in "recipeRequired" and "loginRequired" functions that handle boilerplate?
func deleteRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID, _ := strconv.Atoi(mux.Vars(r)["id"])

	if _, err := recipeByID(recipeID, false); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "No recipe with id=%v exists", recipeID)
		} else {
			apiError(w, http.StatusInternalServerError, "Problem loading recipe", err)
		}
		return
	}

	connect()
	qr := "DELETE FROM recipe WHERE recipe_id = ?"
	ql := "DELETE FROM recipe_label WHERE recipe_id = ?"
	if _, err := db.Exec(qr, recipeID); err != nil {
		apiError(w, http.StatusInternalServerError, "Problem deleting recipe", err)
	}
	if _, err := db.Exec(ql, recipeID); err != nil {
		apiError(w, http.StatusInternalServerError, "Problem deleting recipe-label links", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
