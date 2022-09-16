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

//TODO: How can I do something like python decorators to wrap certain methods
//    in `recipeRequired` or `accessibleToUser` code to minimize duplication?

/* GET */
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
			return
		} else {
			apiError(w, http.StatusInternalServerError, "Problem loading recipe", err)
		}
	} else {
		json.NewEncoder(w).Encode(recipe)
	}
}

func getNotesForRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID, _ := strconv.Atoi(mux.Vars(r)["id"])
	if notes, err := notesByRecipeID(recipeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "No notes for recipe with id=%v exists", recipeID)
			return
		} else {
			apiError(w, http.StatusInternalServerError, "Problem loading notes", err)
		}
	} else {
		json.NewEncoder(w).Encode(notes)
	}
}

/* UPDATE */

/* CREATE */
func tagRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID, _ := strconv.Atoi(mux.Vars(r)["recipe_id"])
	labelID, _ := strconv.Atoi(mux.Vars(r)["label_id"])

	// Make sure we have both recipe and label
	if _, err := recipeByID(recipeID, false); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "No recipe with id=%v exists", recipeID)
			return
		} else {
			apiError(w, http.StatusInternalServerError, "Problem loading recipe", err)
			return
		}
	}
	if _, err := labelByID(labelID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "No label with id=%v exists", labelID)
			return
		} else {
			apiError(w, http.StatusInternalServerError, "Problem loading label", err)
			return
		}
	}
	linked, err := recipeLabelExists(recipeID, labelID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "problem checking recipe-label link", err)
		return
	}

	if linked {
		w.WriteHeader(http.StatusNoContent)
		return
	} else {
		if err := createRecipeLabel(recipeID, labelID); err != nil {
			apiError(w, http.StatusInternalServerError, "problem linking recipe to label", err)
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
}

/* DELETE */
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
