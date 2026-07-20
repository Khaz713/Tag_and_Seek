package main

import (
	"database/sql"
	"net/http"

	"github.com/Khaz713/Tag_and_Seek/internal/database"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/Khaz713/Tag_and_Seek/internal/routing"
)

func (cfg *apiConfig) handlerRegister(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeRequestGob[routing.RegisterRequest](w, r)
	if !ok {
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
	session, err := cfg.db.CreateSession(r.Context(), newUser.ID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	writeResponseGob[routing.RegisterResponse](w, http.StatusCreated, routing.RegisterResponse{
		UserID:   newUser.ID.String(),
		Token:    session.Token.String(),
		Username: newUser.Username,
	})

}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeRequestGob[routing.RegisterRequest](w, r)
	if !ok {
		return
	}

	user, err := cfg.db.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	session, err := cfg.db.CreateSession(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	writeResponseGob[routing.RegisterResponse](w, http.StatusOK, routing.RegisterResponse{
		UserID:   user.ID.String(),
		Token:    session.Token.String(),
		Username: user.Username,
	})

}
