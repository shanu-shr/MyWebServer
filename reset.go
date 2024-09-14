package main

import (
	"net/http"
)

func (cfg *apiConfig) fileServerHitsResteHandler (w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/palin; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}