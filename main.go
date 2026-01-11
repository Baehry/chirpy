package main

import _ "github.com/lib/pq"
import (
	"net/http"
	"fmt"
	"os"
	"sync/atomic"
	"encoding/json"
	"strings"
	"github.com/joho/godotenv"
	"github.com/Baehry/chirpy/internal/database"
	"database/sql"
	"time"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	dbQueries := database.New(db)
	var apiCfg apiConfig
	apiCfg.dbQueries = dbQueries
	apiCfg.platform = os.Getenv("PLATFORM")
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", HealthzHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.MetricsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.ResetHandler)
	mux.HandleFunc("POST /api/validate_chirp", ValidateChirpHandler)
	mux.HandleFunc("POST /api/users", apiCfg.UsersHandler)
	server := http.Server {
		Handler: mux,
		Addr: ":8080",
	}
	err = server.ListenAndServe()
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}

func HealthzHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	writer.Write([]byte("OK"))
}

func (cfg *apiConfig) MetricsHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(200)
	writer.Write([]byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) ResetHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if cfg.platform != "dev" {
		writer.WriteHeader(403)
		return
	}
	writer.WriteHeader(200)
	cfg.fileserverHits.Store(0)
	cfg.dbQueries.ResetUsers(request.Context())
}

func ValidateChirpHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/json; charset=utf-8")
	type parameters struct {
        Body string `json:"body"`
    }
	type errorObj struct {
		Error string `json:"error"`
	}
	type resultCleaned struct {
		CleanedBody string `json:"cleaned_body"`
	}
    decoder := json.NewDecoder(request.Body)
    var params parameters
    decoder.Decode(&params)
	if len(params.Body) > 140 {
		errObj := errorObj{
			Error: "Chirp is too long",
		}
		dat, _ := json.Marshal(errObj)
		writer.WriteHeader(400)
		writer.Write(dat)
		return
	}
	splitString := strings.Split(params.Body, " ")
	for i, word := range splitString {
		if strings.ToLower(word) == "kerfuffle" || strings.ToLower(word) == "sharbert" || strings.ToLower(word) == "fornax" {
			splitString[i] = "****"
		}
	}
	resCleaned := resultCleaned{
		CleanedBody: strings.Join(splitString, " "),
	}
	dat, _ := json.Marshal(resCleaned)
	writer.WriteHeader(200)
	writer.Write(dat)
	return
}

func (cfg *apiConfig) UsersHandler(writer http.ResponseWriter, request *http.Request) {
	type parameters struct {
        Email string `json:"email"`
    }
	type User struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}
	decoder := json.NewDecoder(request.Body)
    var params parameters
    decoder.Decode(&params)
	user, err := cfg.dbQueries.CreateUser(request.Context(), params.Email)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	actualUser := User{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}
	dat, _ := json.Marshal(actualUser)
	writer.WriteHeader(201)
	writer.Write(dat)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}