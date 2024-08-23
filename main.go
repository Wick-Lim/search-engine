package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/blevesearch/bleve/v2"
	_ "github.com/lib/pq"
)

var index bleve.Index
var db *sql.DB

func main() {
	var err error

	connStr := os.Getenv("POSTGRES_CONN")
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	indexPath := ".index"
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		index, err = bleve.New(indexPath, bleve.NewIndexMapping())
		if err != nil {
			log.Fatalf("Failed to create index: %v", err)
		}
	} else {
		index, err = bleve.Open(indexPath)
		if err != nil {
			log.Fatalf("Failed to open index: %v", err)
		}
	}
	defer index.Close()

	http.HandleFunc("/", heartbeatHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/insert", insertHandler)

	fmt.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func insertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var id int
	err := db.QueryRow("INSERT INTO documents(content) VALUES($1) RETURNING id", req.Content).Scan(&id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to insert data: %v", err), http.StatusInternalServerError)
		return
	}

	err = index.Index(strconv.Itoa(id), req.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to index data: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Document inserted with ID: %d", id)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	if index == nil {
		http.Error(w, "Index is not initialized", http.StatusInternalServerError)
		return
	}

	queryParam := r.URL.Query().Get("q")
	if queryParam == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	query := bleve.NewMatchQuery(queryParam)
	searchRequest := bleve.NewSearchRequest(query)
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(searchResult); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}
