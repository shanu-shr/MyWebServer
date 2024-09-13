package main

import (
	"fmt"
	"log"
	"net/http"
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

func myCustomHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) fileServerHitsLoggerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`
<html>

<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
</body>

</html>
	`, cfg.fileserverHits)))
}


func (cfg *apiConfig) fileServerHitsResteHandler (w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/palin; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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

	srv := &http.Server{
		Addr : ":"+port,
		Handler : mux,
	}

	log.Printf("Serving files from %s on port %s\n", filePathRoot, port)
	log.Fatal(srv.ListenAndServe())
}

