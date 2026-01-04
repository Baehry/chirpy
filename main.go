package main

import (
	"net/http"
	"fmt"
	"os"
	"sync/atomic"
	"encoding/json"
	"strings"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	var apiCfg apiConfig
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", HealthzHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.MetricsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.ResetHandler)
	mux.HandleFunc("POST /api/validate_chirp", ValidateChirpHandler)
	server := http.Server {
		Handler: mux,
		Addr: ":8080",
	}
	err := server.ListenAndServe()
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
	writer.WriteHeader(200)
	cfg.fileserverHits.Store(0)
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

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}