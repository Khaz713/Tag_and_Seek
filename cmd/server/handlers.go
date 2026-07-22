package main

import (
	"database/sql"
	"math/rand"
	"net/http"

	"github.com/Khaz713/Tag_and_Seek/internal/database"
	"github.com/Khaz713/Tag_and_Seek/internal/gamelogic"
	"github.com/google/uuid"
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

func (cfg *apiConfig) handlerLogout(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeRequestGob[routing.LogoutRequest](w, r)
	if !ok {
		return
	}
	token, err := uuid.Parse(req.Token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	err = cfg.db.DeleteSessionByToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (cfg *apiConfig) handlerGameHistory(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeRequestGob[routing.TokenIdentification](w, r)
	if !ok {
		return
	}
	sessionUUID, err := uuid.Parse(req.Token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	session, err := cfg.db.GetSessionByToken(r.Context(), sessionUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	games, err := cfg.db.GetGamesByUserID(r.Context(), session.UserID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var history []routing.GameHistory

	for _, game := range games {
		gamePlayers, err := cfg.db.GetParticipantsByGameID(r.Context(), game.ID)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		var players []routing.GamePlayers
		for _, player := range gamePlayers {
			players = append(players, routing.GamePlayers{
				UserID:        player.UserID.String(),
				Username:      player.Username,
				HiddenSeconds: int(player.HiddenSeconds),
				Ranking:       int(player.Ranking),
			})
		}
		history = append(history, routing.GameHistory{
			GameID:          game.ID.String(),
			MapIndex:        int(game.MapIndex),
			WinnerID:        game.WinnerID.UUID.String(),
			DurationSeconds: int(game.DurationSeconds),
			PlayedAt:        game.PlayedAt,
			Players:         players,
		})
	}
	writeResponseGob[routing.GameHistoryResponse](w, http.StatusOK, routing.GameHistoryResponse{
		Games: history,
	})
}

func (cfg *apiConfig) handlerCreateRoom(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeRequestGob[routing.CreateRoomRequest](w, r)
	if !ok {
		return
	}
	sessionUUID, err := uuid.Parse(req.Token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	session, err := cfg.db.GetSessionByToken(r.Context(), sessionUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	cfg.roomsMux.Lock()
	defer cfg.roomsMux.Unlock()
	_, ok = cfg.rooms[req.Name]
	if ok {
		http.Error(w, "Room already exists", http.StatusConflict)
		return
	}
	mapIndex := rand.Intn(len(gamelogic.Maps))
	newRoom := gamelogic.GameRoom{
		ID:         uuid.New().String(),
		MapIndex:   mapIndex,
		Map:        gamelogic.Maps[mapIndex],
		Players:    make(map[string]*gamelogic.ServerPlayer),
		State:      routing.StateLobby,
		GameWinner: "",
		Size:       req.Size,
		Round:      0,
	}
	newRoom.AddPlayer(session.UserID.String())
	cfg.rooms[req.Name] = &newRoom

	writeResponseGob[routing.CreateRoomResponse](w, http.StatusCreated, routing.CreateRoomResponse{
		RoomID: newRoom.ID,
	})
	//TODO make new rabbit and give client info to join
}

func (cfg *apiConfig) handlerGetRooms(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeRequestGob[routing.TokenIdentification](w, r)
	if !ok {
		return
	}
	sessionUUID, err := uuid.Parse(req.Token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	_, err = cfg.db.GetSessionByToken(r.Context(), sessionUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	roomsInfo := []routing.RoomInfo{}
	cfg.roomsMux.RLock()
	for name, room := range cfg.rooms {
		if room.State == routing.StateLobby {
			roomsInfo = append(roomsInfo, routing.RoomInfo{
				ID:         room.ID,
				Name:       name,
				Size:       room.Size,
				PlayersNum: len(room.Players),
			})
		}
	}
	writeResponseGob[routing.GetRoomsResponse](w, http.StatusOK, routing.GetRoomsResponse{
		Rooms: roomsInfo,
	})
}

func (cfg *apiConfig) handlerJoinRoom(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeRequestGob[routing.JoinRoomRequest](w, r)
	if !ok {
		return
	}
	sessionUUID, err := uuid.Parse(req.Token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	session, err := cfg.db.GetSessionByToken(r.Context(), sessionUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	cfg.roomsMux.Lock()
	defer cfg.roomsMux.Unlock()
	room, ok := cfg.rooms[req.RoomName]
	if !ok {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}
	if room.State != routing.StateLobby {
		http.Error(w, "Game already in progress", http.StatusForbidden)
		return
	}
	if len(room.Players) >= room.Size {
		http.Error(w, "Room if full", http.StatusConflict)
		return
	}
	room.AddPlayer(session.UserID.String())

	writeResponseGob[routing.CreateRoomResponse](w, http.StatusOK, routing.CreateRoomResponse{
		RoomID: room.ID, //temp response struct
	})
	//TODO give client info of rabbit queue to join
}
