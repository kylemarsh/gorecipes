package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

//TODO: How can I do something like python decorators to wrap certain methods
//    in `recipeRequired` or `accessibleToUser` code to minimize duplication?

/* GET */
func getAllRecipes(w http.ResponseWriter, r *http.Request) {
	recipes, err := activeRecipes(true)

	if err != nil {
		apiError(w, http.StatusInternalServerError, "Problem loading recipes", err)
		return
	}
	json.NewEncoder(w).Encode(recipes)
}

func getRecipeByID(w http.ResponseWriter, r *http.Request) {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "recipe ID must be an integer", err)
		return
	}

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
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "recipe ID must be an integer", err)
		return
	}
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
func updateExistingRecipe(w http.ResponseWriter, r *http.Request) {
	recipeId, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "recipe ID must be an integer", err)
		return
	}
	title := r.FormValue("title")
	if title == "" {
		apiError(w, http.StatusBadRequest, "title is required", nil)
		return
	}
	activeTime, err := strconv.Atoi(r.FormValue("activeTime"))
	if err != nil {
		apiError(w, http.StatusBadRequest, "activeTime must be an integer", err)
		return
	}
	totalTime, err := strconv.Atoi(r.FormValue("totalTime"))
	if err != nil {
		apiError(w, http.StatusBadRequest, "totalTime must be an integer", err)
		return
	}
	body := r.FormValue("body")

	err = updateRecipe(recipeId, title, body, activeTime, totalTime)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not create recipe", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func flagNote(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "note ID must be an integer", err)
		return
	}

	if _, err := getNoteByID(noteID); err != nil {
		apiError(w, http.StatusNotFound, "note does not exist", err)
		return
	}
	if err := setNoteFlag(noteID, true); err != nil {
		apiError(w, http.StatusInternalServerError, "problem flagging note", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func unFlagNote(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "note ID must be an integer", err)
		return
	}

	if _, err := getNoteByID(noteID); err != nil {
		apiError(w, http.StatusNotFound, "note does not exist", err)
		return
	}
	if err := setNoteFlag(noteID, false); err != nil {
		apiError(w, http.StatusInternalServerError, "problem flagging note", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func editNote(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "note ID must be an integer", err)
		return
	}
	noteText := r.FormValue("text")

	if _, err := getNoteByID(noteID); err != nil {
		apiError(w, http.StatusNotFound, "note does not exist", err)
		return
	}
	if err := setNoteText(noteID, noteText); err != nil {
		apiError(w, http.StatusInternalServerError, "problem updating note", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

/* CREATE */
func createNewRecipe(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	if title == "" {
		apiError(w, http.StatusBadRequest, "title is required", nil)
		return
	}
	activeTime, err := strconv.Atoi(r.FormValue("activeTime"))
	if err != nil {
		apiError(w, http.StatusBadRequest, "activeTime must be an integer", err)
		return
	}
	totalTime, err := strconv.Atoi(r.FormValue("totalTime"))
	if err != nil {
		apiError(w, http.StatusBadRequest, "totalTime must be an integer", err)
		return
	}
	body := r.FormValue("body")

	recipe, err := createRecipe(title, body, activeTime, totalTime)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not create recipe", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recipe)
}

func createNoteOnRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "recipe ID must be an integer", err)
		return
	}
	noteText := r.FormValue("text")

	// Validate that the recipe exists
	if _, err := recipeByID(recipeID, false); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			apiError(w, http.StatusNotFound, "recipe does not exist", err)
		} else {

			apiError(w, http.StatusInternalServerError, "Problem loading recipe", err)
		}
		return
	}

	note, err := createNote(recipeID, noteText)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "problem creating note", err)
		return
	}
	json.NewEncoder(w).Encode(note)
}

func tagRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID, err := strconv.Atoi(mux.Vars(r)["recipe_id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "recipe ID must be an integer", err)
		return
	}
	labelID, err := strconv.Atoi(mux.Vars(r)["label_id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "label ID must be an integer", err)
		return
	}

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

func addLabel(w http.ResponseWriter, r *http.Request) {
	labelName := strings.ToLower(mux.Vars(r)["label_name"])
	label, err := labelByName(labelName)
	if err == nil { // No error means the label alredy exists
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(label)
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		// ErrNoRows means the label doesn't yet exist; anything else is actually an error
		apiError(w, http.StatusInternalServerError, "problem checking label", err)
		return
	}
	label, err = createLabel(labelName)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "problem creating label", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(label)
}

/* DELETE */
func deleteRecipeHard(w http.ResponseWriter, r *http.Request) {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "recipe ID must be an integer", err)
		return
	}

	_, err = recipeByID(recipeID, false)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		apiError(w, http.StatusInternalServerError, "Problem loading recipe", err)
		return
	}

	connect()
	qr := "DELETE FROM recipe WHERE recipe_id = ?"
	ql := "DELETE FROM recipe_label WHERE recipe_id = ?"
	qn := "DELETE FROM note WHERE recipe_id = ?"
	if _, err := db.Exec(qr, recipeID); err != nil {
		apiError(w, http.StatusInternalServerError, "Problem deleting recipe", err)
		return
	}
	if _, err := db.Exec(ql, recipeID); err != nil {
		apiError(w, http.StatusInternalServerError, "Problem deleting recipe-label links", err)
		return
	}
	if _, err := db.Exec(qn, recipeID); err != nil {
		apiError(w, http.StatusInternalServerError, "Problem deleting notes", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteRecipeSoft(w http.ResponseWriter, r *http.Request) {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "recipe ID must be an integer", err)
		return
	}
	err = softDeleteRecipe(recipeID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not soft-delete recipe", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func recipeRestore(w http.ResponseWriter, r *http.Request) {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "recipe ID must be an integer", err)
		return
	}
	err = unDeleteRecipe(recipeID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not un-delete recipe", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func untagRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID, err := strconv.Atoi(mux.Vars(r)["recipe_id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "recipe ID must be an integer", err)
		return
	}
	labelID, err := strconv.Atoi(mux.Vars(r)["label_id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "label ID must be an integer", err)
		return
	}

	if err := deleteRecipeLabel(recipeID, labelID); err != nil {
		apiError(w, http.StatusInternalServerError, "problem deleting recipe-label link", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func removeNote(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		apiError(w, http.StatusBadRequest, "note ID must be an integer", err)
		return
	}

	if err := deleteNote(noteID); err != nil {
		apiError(w, http.StatusInternalServerError, "problem deleting note", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
