package main

import (
	"io"
	"net/http"

	"github.com/Khaz713/Tag_and_Seek/internal/database"
	pubsub2 "github.com/Khaz713/Tag_and_Seek/internal/pubsub"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/Khaz713/Tag_and_Seek/internal/routing"
)

func (cfg *apiConfig) handlerRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	req, err := pubsub2.DecodeGob[routing.RegisterRequest](bodyBytes)
	if err != nil {
		http.Error(w, "Failed to decode body", http.StatusBadRequest)
		return
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to generate password hash", http.StatusInternalServerError)
		return
	}
	newUser, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Username:     req.Username,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { //unique_violation code
				http.Error(w, "User already exists", http.StatusConflict)
				return
			}
		}
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	resp, err := pubsub2.EncodeGob(routing.RegisterResponse{
		UserId:   newUser.ID.String(),
		Username: newUser.Username,
	})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(resp)
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}

}
