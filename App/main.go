package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Response struct {
	UserId int `json:"userId"`
}

var db *sql.DB
var wg sync.WaitGroup

func main() {
	dbName, _ := os.LookupEnv("DB_NAME")
	dbUrl, _ := os.LookupEnv("POSTGRES_URL")
	dbUser, _ := os.LookupEnv("POSTGRES_USERNAME")
	dbPassword, _ := os.LookupEnv("POSTGRES_PASSWORD")

	connString := fmt.Sprintf("host=%s port=5432 user=%s dbname=%s password=%s sslmode=disable",
		dbUrl, dbUser, dbName, dbPassword)

	db, _ = sql.Open("postgres", connString)

	defer db.Close()

	router := mux.NewRouter()

	router.HandleFunc("/api/v1/Autorizacao", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Requisição recebida")
		ProcessRequestConcurrently(w, r)
	})

	log.Println("Iniciando API")

	log.Fatal(http.ListenAndServe(":8080", router))
}

func ProcessRequestConcurrently(w http.ResponseWriter, r *http.Request) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		authorizeHeader := r.Header.Get("Authorize")

		if authorizeHeader == "" {
			response, _ := json.Marshal(Response{UserId: 0})
			http.Error(w, string(response), http.StatusForbidden)
			return
		}

		query := "SELECT fk_idusuario FROM Sessao WHERE hashsessao = $1"

		apiKey := strings.Replace(authorizeHeader, "Bearer ", "", 1)

		rows, err := db.Query(query, apiKey)
		if err != nil {
			log.Fatal(err)
			return
		}
		defer rows.Close()

		query = "UPDATE sessao SET ultimoacesso = $1 WHERE hashsessao = $2"

		_, _ = db.Exec(query, time.Now(), apiKey)

		if rows.Next() {
			var userId int
			err := rows.Scan(&userId)
			if err != nil {
				log.Fatal(err)
				return
			}
			response, _ := json.Marshal(Response{UserId: userId})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(response)
		} else {
			response, _ := json.Marshal(Response{UserId: 0})
			http.Error(w, string(response), http.StatusForbidden)
		}
	}()
	wg.Wait()
}
