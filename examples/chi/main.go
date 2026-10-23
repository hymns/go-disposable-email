// Sign-up with chi. Handlers are plain net/http, so the check is the same.
//
//	go run . &
//	curl -i -d '{"email":"a@mailinator.com"}' localhost:8080/register   # 400
//	curl -i -d '{"email":"a@gmail.com"}' localhost:8080/register        # 201
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	disposable "github.com/hymns/go-disposable-email"
)

func main() {
	r := chi.NewRouter()
	r.Post("/register", register)
	log.Fatal(http.ListenAndServe(":8080", r))
}

func register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := disposable.Check(req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// create the account
	w.WriteHeader(http.StatusCreated)
}
