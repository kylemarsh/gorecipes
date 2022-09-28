package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
)

// Authentication Middleware. Paths under this router require valid
// authentication to access
func authRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var header = r.Header.Get("x-access-token")
		tokenString := strings.TrimSpace(header)
		if tokenString == "" {
			msg := "missing auth token"
			code := http.StatusUnauthorized
			http.Error(w, msg, code)
			fmt.Printf("%d: %v\n", code, msg)
			return
		}

		_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(conf.JwtSecret), nil
		})

		if err != nil {
			errstring := err.Error()
			var msg string
			var code int
			if errstring == "Token is expired" {
				msg = "auth token expired; please log in again"
				code = http.StatusUnauthorized
			} else {
				msg = "invalid auth token"
				code = http.StatusBadRequest
			}
			http.Error(w, msg, code)
			fmt.Printf("%d: %v\n", code, msg)
			return
		}
		next.ServeHTTP(w, r)
	})
}

//TODO: How can I do something like python decorators to wrap certain methods
//    in `recipeRequired` or `accessibleToUser` code to minimize duplication?

/* GET */
func getAllRecipes(w http.ResponseWriter, r *http.Request) *appError {
	recipes, err := activeRecipes(true)

	if err != nil {
		return &appError{http.StatusInternalServerError, "Problem loading recipes", err}
	}
	json.NewEncoder(w).Encode(recipes)
	return nil
}

func getRecipeByID(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}

	if recipe, err := recipeByID(recipeID, true); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "No recipe with id=%v exists", recipeID)
			return nil
		} else {
			return &appError{http.StatusInternalServerError, "Problem loading recipe", err}
		}
	} else {
		json.NewEncoder(w).Encode(recipe)
	}
	return nil
}

func getNotesForRecipe(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}
	if notes, err := notesByRecipeID(recipeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "No notes for recipe with id=%v exists", recipeID)
			return nil
		} else {
			return &appError{http.StatusInternalServerError, "Problem loading notes", err}
		}
	} else {
		json.NewEncoder(w).Encode(notes)
		return nil
	}
}

/* UPDATE */
func updateExistingRecipe(w http.ResponseWriter, r *http.Request) *appError {
	recipeId, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}
	title := r.FormValue("title")
	if title == "" {
		return &appError{http.StatusBadRequest, "title is required", nil}
	}
	activeTime, err := strconv.Atoi(r.FormValue("activeTime"))
	if err != nil {
		return &appError{http.StatusBadRequest, "activeTime must be an integer", err}
	}
	totalTime, err := strconv.Atoi(r.FormValue("totalTime"))
	if err != nil {
		return &appError{http.StatusBadRequest, "totalTime must be an integer", err}
	}
	body := r.FormValue("body")

	err = updateRecipe(recipeId, title, body, activeTime, totalTime)
	if err != nil {
		return &appError{http.StatusInternalServerError, "could not create recipe", err}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func flagNote(w http.ResponseWriter, r *http.Request) *appError {
	noteID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "note ID must be an integer", err}
	}

	if _, err := getNoteByID(noteID); err != nil {
		return &appError{http.StatusNotFound, "note does not exist", err}
	}
	if err := setNoteFlag(noteID, true); err != nil {
		return &appError{http.StatusInternalServerError, "problem flagging note", err}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func unFlagNote(w http.ResponseWriter, r *http.Request) *appError {
	noteID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "note ID must be an integer", err}
	}

	if _, err := getNoteByID(noteID); err != nil {
		return &appError{http.StatusNotFound, "note does not exist", err}
	}
	if err := setNoteFlag(noteID, false); err != nil {
		return &appError{http.StatusInternalServerError, "problem flagging note", err}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func editNote(w http.ResponseWriter, r *http.Request) *appError {
	noteID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "note ID must be an integer", err}
	}
	noteText := r.FormValue("text")

	if _, err := getNoteByID(noteID); err != nil {
		return &appError{http.StatusNotFound, "note does not exist", err}
	}
	if err := setNoteText(noteID, noteText); err != nil {
		return &appError{http.StatusInternalServerError, "problem updating note", err}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

/* CREATE */
func createNewRecipe(w http.ResponseWriter, r *http.Request) *appError {
	title := r.FormValue("title")
	if title == "" {
		return &appError{http.StatusBadRequest, "title is required", nil}
	}
	activeTime, err := strconv.Atoi(r.FormValue("activeTime"))
	if err != nil {
		return &appError{http.StatusBadRequest, "activeTime must be an integer", err}
	}
	totalTime, err := strconv.Atoi(r.FormValue("totalTime"))
	if err != nil {
		return &appError{http.StatusBadRequest, "totalTime must be an integer", err}
	}
	body := r.FormValue("body")

	recipe, err := createRecipe(title, body, activeTime, totalTime)
	if err != nil {
		return &appError{http.StatusInternalServerError, "could not create recipe", err}
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recipe)
	return nil
}

func createNoteOnRecipe(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}
	noteText := r.FormValue("text")

	// Validate that the recipe exists
	if _, err := recipeByID(recipeID, false); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &appError{http.StatusNotFound, "recipe does not exist", err}
		} else {
			return &appError{http.StatusInternalServerError, "Problem loading recipe", err}
		}
	}

	note, err := createNote(recipeID, noteText)
	if err != nil {
		return &appError{http.StatusInternalServerError, "problem creating note", err}
	}
	json.NewEncoder(w).Encode(note)
	return nil
}

func tagRecipe(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, err := strconv.Atoi(mux.Vars(r)["recipe_id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}
	labelID, err := strconv.Atoi(mux.Vars(r)["label_id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "label ID must be an integer", err}
	}

	// Make sure we have both recipe and label
	if _, err := recipeByID(recipeID, false); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			msg := fmt.Sprintf("No recipe with id=%v exists", recipeID)
			return &appError{http.StatusNotFound, msg, err}
		} else {
			return &appError{http.StatusInternalServerError, "Problem loading recipe", err}
		}
	}
	if _, err := labelByID(labelID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			msg := fmt.Sprintf("No label with id=%v exists", labelID)
			return &appError{http.StatusNotFound, msg, err}
		} else {
			return &appError{http.StatusInternalServerError, "Problem loading label", err}
		}
	}
	linked, err := recipeLabelExists(recipeID, labelID)
	if err != nil {
		return &appError{http.StatusInternalServerError, "problem checking recipe-label link", err}
	}

	if linked {
		w.WriteHeader(http.StatusNoContent)
		return nil
	} else {
		if err := createRecipeLabel(recipeID, labelID); err != nil {
			return &appError{http.StatusInternalServerError, "problem linking recipe to label", err}
		}
	}
	w.WriteHeader(http.StatusCreated)
	return nil
}

func addLabel(w http.ResponseWriter, r *http.Request) *appError {
	labelName := strings.ToLower(mux.Vars(r)["label_name"])
	label, err := labelByName(labelName)
	if err == nil { // No error means the label alredy exists
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(label)
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		// ErrNoRows means the label doesn't yet exist; anything else is actually an error
		return &appError{http.StatusInternalServerError, "problem checking label", err}
	}
	label, err = createLabel(labelName)
	if err != nil {
		return &appError{http.StatusInternalServerError, "problem creating label", err}
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(label)
	return nil
}

/* DELETE */
func deleteRecipeHard(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}

	_, err = recipeByID(recipeID, false)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return &appError{http.StatusInternalServerError, "Problem loading recipe", err}
	}

	connect()
	qr := "DELETE FROM recipe WHERE recipe_id = ?"
	ql := "DELETE FROM recipe_label WHERE recipe_id = ?"
	qn := "DELETE FROM note WHERE recipe_id = ?"
	if _, err := db.Exec(qr, recipeID); err != nil {
		return &appError{http.StatusInternalServerError, "Problem deleting recipe", err}
	}
	if _, err := db.Exec(ql, recipeID); err != nil {
		return &appError{http.StatusInternalServerError, "Problem deleting recipe-label links", err}
	}
	if _, err := db.Exec(qn, recipeID); err != nil {
		return &appError{http.StatusInternalServerError, "Problem deleting notes", err}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func deleteRecipeSoft(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}
	err = softDeleteRecipe(recipeID)
	if err != nil {
		return &appError{http.StatusInternalServerError, "could not soft-delete recipe", err}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func recipeRestore(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}
	err = unDeleteRecipe(recipeID)
	if err != nil {
		return &appError{http.StatusInternalServerError, "could not un-delete recipe", err}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func untagRecipe(w http.ResponseWriter, r *http.Request) *appError {
	recipeID, err := strconv.Atoi(mux.Vars(r)["recipe_id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "recipe ID must be an integer", err}
	}
	labelID, err := strconv.Atoi(mux.Vars(r)["label_id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "label ID must be an integer", err}
	}

	if err := deleteRecipeLabel(recipeID, labelID); err != nil {
		return &appError{http.StatusInternalServerError, "problem deleting recipe-label link", err}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func removeNote(w http.ResponseWriter, r *http.Request) *appError {
	noteID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return &appError{http.StatusBadRequest, "note ID must be an integer", err}
	}

	if err := deleteNote(noteID); err != nil {
		return &appError{http.StatusInternalServerError, "problem deleting note", err}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
