package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Khaz713/Tag_and_Seek/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type apiConfig struct {
	db *database.Queries
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("dbURL")
	port := os.Getenv("serverPort")
	connStr := os.Getenv("connStr")
	cfg := &apiConfig{}

	fmt.Println("Connecting to RabbitMQ...")
	conn, err := amqp.Dial(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	fmt.Println("Connected to RabbitMQ")

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

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register", cfg.handlerRegister)
	mux.HandleFunc("POST /api/login", cfg.handlerLogin)
	mux.HandleFunc("POST /api/logout", cfg.handlerLogout)

	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: mux,
	}
	log.Fatal(srv.ListenAndServe())
}
