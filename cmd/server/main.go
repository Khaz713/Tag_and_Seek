package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Khaz713/Tag_and_Seek/internal/database"
	"github.com/Khaz713/Tag_and_Seek/internal/gamelogic"
	"github.com/Khaz713/Tag_and_Seek/internal/pubsub"
	"github.com/Khaz713/Tag_and_Seek/internal/routing"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type apiConfig struct {
	db         *database.Queries
	dbPool     *sql.DB
	roomsMux   sync.RWMutex
	rooms      map[string]*gamelogic.GameRoom
	channelMux sync.RWMutex
	channel    *amqp.Channel
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("dbURL")
	port := os.Getenv("serverPort")
	connStr := os.Getenv("connStr")
	cfg := &apiConfig{
		rooms: make(map[string]*gamelogic.GameRoom),
	}

	fmt.Println("Connecting to RabbitMQ...")
	conn, err := amqp.Dial(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	fmt.Println("Connected to RabbitMQ")

	fmt.Println("Opening channel...")
	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	cfg.channel = channel
	defer channel.Close()
	fmt.Println("Connected to channel")
	fmt.Println("Declaring queue...")
	_, err = channel.QueueDeclare(
		routing.GameResultQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-queue-type": "quorum",
		},
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}
	fmt.Println("Declared queue")

	fmt.Println("Connecting to database ...")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("Connected to database")
	cfg.db = database.New(db)
	cfg.dbPool = db

	//background sessions cleaner
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		err := cfg.db.DeleteExpiredSessions(context.Background())
		if err != nil {
			log.Printf("Failed to delete expired sessions: %v", err)
		}

		for range ticker.C {
			err := cfg.db.DeleteExpiredSessions(context.Background())
			if err != nil {
				log.Printf("Failed to delete expired sessions: %v", err)
			}
		}
	}()

	err = pubsub.SubscribeGameResult(conn, func(result routing.GameResult) error {
		gameId, err := uuid.Parse(result.GameID)
		if err != nil {
			return err
		}
		winnerId, err := uuid.Parse(result.WinnerID)
		if err != nil {
			return err
		}
		tx, err := cfg.dbPool.BeginTx(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()

		qtx := cfg.db.WithTx(tx)

		_, err = qtx.CreateGame(context.Background(), database.CreateGameParams{
			ID:              gameId,
			MapIndex:        int32(result.MapIndex),
			WinnerID:        uuid.NullUUID{UUID: winnerId},
			DurationSeconds: int32(result.Duration),
		})
		for _, player := range result.Players {
			userId, err := uuid.Parse(player.PlayerID)
			if err != nil {
				return err
			}
			_, err = qtx.CreateGameUser(context.Background(), database.CreateGameUserParams{
				GameID:        gameId,
				UserID:        userId,
				HiddenSeconds: int32(player.Duration),
				Ranking:       int32(player.Ranking),
			})
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		return nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register", cfg.handlerRegister)
	mux.HandleFunc("POST /api/login", cfg.handlerLogin)
	mux.HandleFunc("POST /api/logout", cfg.handlerLogout)
	mux.HandleFunc("POST /api/history", cfg.handlerGameHistory)
	mux.HandleFunc("POST /api/createRoom", cfg.handlerCreateRoom)
	mux.HandleFunc("POST /api/getRooms", cfg.handlerGetRooms)
	mux.HandleFunc("POST /api/joinRoom", cfg.handlerJoinRoom)

	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: mux,
	}
	log.Fatal(srv.ListenAndServe())
}
