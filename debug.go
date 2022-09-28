package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Debug Mode Middleware -- prohibits accessing certain routes when debug mode is disabled
func debugRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !conf.Debug {
			http.Error(w, "token validation only available for debugging", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validateJwt(w http.ResponseWriter, r *http.Request) *appError {

	var header = r.Header.Get("x-access-token")
	tokenString := strings.TrimSpace(header)
	err := jwtValidate(tokenString)
	if err != nil {
		return &appError{http.StatusBadRequest, "invalid auth token", err}
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func getHash(w http.ResponseWriter, r *http.Request) *appError {

	var password = r.FormValue("password")
	hash, err := hashPassword(password)
	if err != nil {
		return &appError{http.StatusInternalServerError, "problem hashing password", err}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"hash": hash})
	return nil
}

func getJwt(w http.ResponseWriter, r *http.Request) *appError {

	tokenStr, err := jwtGenerate()
	if err != nil {
		return &appError{http.StatusInternalServerError, "could not sign token", err}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenStr})
	return nil
}
