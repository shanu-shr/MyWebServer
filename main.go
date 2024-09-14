package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type apiConfig struct {
	fileserverHits int
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler{
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request){
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Body string `json:"body"`
	}

	type returnVals struct {
		Cleaned_body string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	var params parameters

	err := decoder.Decode(&params)
	if err!= nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	const maxChirpLength = 140
	if len(params.Body)>maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	msg := removeProfaneWords(params.Body)
	respondWithJson(w, http.StatusOK, returnVals{
		Cleaned_body: msg,
	})
}

func main(){
	log.Printf("Starting the server")
	const port = "8080"
	const filePathRoot = "."

	mux := http.NewServeMux()
	apicfg := apiConfig{}

	mux.Handle("/app/", http.StripPrefix("/app", apicfg.middlewareMetricsInc(http.FileServer(http.Dir(filePathRoot)))))
	mux.Handle("GET /api/healthz", http.HandlerFunc(myCustomHandler))
	mux.Handle("GET /admin/metrics", http.HandlerFunc(apicfg.fileServerHitsLoggerHandler))
	mux.Handle("GET /api/reset", http.HandlerFunc(apicfg.fileServerHitsResteHandler))
	mux.Handle("POST /api/validate_chirp", http.HandlerFunc(validateChirpHandler))

	srv := &http.Server{
		Addr : ":"+port,
		Handler : mux,
	}

	log.Printf("Serving files from %s on port %s\n", filePathRoot, port)
	log.Fatal(srv.ListenAndServe())
}

//Helper functions

func removeProfaneWords(msg string) string {
	words := strings.Split(msg, " ")
	temp := ""

	for i := 0; i <len(words); i++ {
		temp = strings.ToLower(words[i])
		if temp == "kerfuffle" || temp == "sharbert" || temp == "fornax" {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		log.Printf("Responding with 5xx error %s", msg)
	}

	type errorResponse struct {
		Error string `json:"error"`
	}

	respondWithJson(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")

	data,err := json.Marshal(payload)
	if err!= nil {
		log.Printf("Error mashalling Json %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(code)
	w.Write(data)
}