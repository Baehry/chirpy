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
	"github.com/google/uuid"
	"github.com/Baehry/chirpy/internal/auth"
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
	mux.HandleFunc("POST /api/users", apiCfg.UsersHandler)
	mux.HandleFunc("POST /api/chirps", apiCfg.ChirpsHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.GetChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.GetChirpHandler)
	mux.HandleFunc("POST /api/login", apiCfg.LoginHandler)
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

func (cfg *apiConfig) ChirpsHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/json; charset=utf-8")
	type parameters struct {
        Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
    }
	type errorObj struct {
		Error string `json:"error"`
	}
    decoder := json.NewDecoder(request.Body)
    var params parameters
    if err := decoder.Decode(&params); err != nil {
		errObj := errorObj {
			Error: err.Error(),
		}
		dat, _ := json.Marshal(errObj)
		writer.WriteHeader(500)
		writer.Write(dat)
	}
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
	result, err := cfg.dbQueries.CreateChirp(request.Context(), database.CreateChirpParams{
		Body: strings.Join(splitString, " "),
		UserID: params.UserID,
	})
	if err != nil {
		errObj := errorObj {
			Error: err.Error(),
		}
		dat, _ := json.Marshal(errObj)
		writer.WriteHeader(500)
		writer.Write(dat)
	}
	dat, err := json.Marshal(result)
	if err != nil {
		errObj := errorObj {
			Error: err.Error(),
		}
		dat, _ := json.Marshal(errObj)
		writer.WriteHeader(500)
		writer.Write(dat)
	}
	writer.WriteHeader(201)
	writer.Write(dat)
	return
}

func (cfg *apiConfig) UsersHandler(writer http.ResponseWriter, request *http.Request) {
	type parameters struct {
		Password string `json:"password"`
        Email string `json:"email"`
    }
	decoder := json.NewDecoder(request.Body)
    var params parameters
    decoder.Decode(&params)
	hashedPassword, _ := auth.HashPassword(params.Password)
	user, err := cfg.dbQueries.CreateUser(request.Context(), database.CreateUserParams{
		Email: params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	dat, _ := json.Marshal(user)
	writer.WriteHeader(201)
	writer.Write(dat)
}

func (cfg *apiConfig) GetChirpsHandler(writer http.ResponseWriter, request *http.Request) {
	result, _ := cfg.dbQueries.GetAllChirps(request.Context())
	dat, _ := json.Marshal(result)
	writer.WriteHeader(200)
	writer.Write(dat)
}

func (cfg *apiConfig) GetChirpHandler(writer http.ResponseWriter, request *http.Request) {
	id, err := uuid.Parse(request.PathValue("chirpID"))
	if err != nil {
		writer.WriteHeader(404)
		return
	}
	result, err := cfg.dbQueries.GetChirp(request.Context(), id)
	if err != nil {
		writer.WriteHeader(404)
		return
	}
	dat, _ := json.Marshal(result)
	writer.WriteHeader(200)
	writer.Write(dat)
}

func (cfg *apiConfig) LoginHandler(writer http.ResponseWriter, request *http.Request) {
	type parameters struct {
		Password string `json:"password"`
        Email string `json:"email"`
    }
	decoder := json.NewDecoder(request.Body)
    var params parameters
    decoder.Decode(&params)
	user, err := cfg.dbQueries.GetUserByEmail(request.Context(), params.Email)
	if err != nil {
		writer.WriteHeader(401)
		writer.Write([]byte("Incorrect email or password"))
		return
	}
	if valid, err := auth.CheckPasswordHash(params.Password, user.HashedPassword); (!valid) || (err != nil) {
		writer.WriteHeader(401)
		writer.Write([]byte("Incorrect email or password"))
		return
	}
	dat, err := json.Marshal(user)
	if err != nil {
		writer.WriteHeader(500)
		writer.Write([]byte(err.Error()))
	}
	writer.WriteHeader(200)
	writer.Write(dat)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}