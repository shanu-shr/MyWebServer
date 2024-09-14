package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	database "github.com/shanu-shr/goserver/Database"
	"golang.org/x/crypto/bcrypt"
)

type apiConfig struct {
	fileserverHits int
	db *database.DB
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler{
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request){
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) validateChirpHandler(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Body string `json:"body"`
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

	chirp,_ := cfg.db.CreateChirp(msg)
	respondWithJson(w, http.StatusCreated, chirp)
}

func (cfg* apiConfig) getChirpHandler(w http.ResponseWriter, r *http.Request){
	chirps, _ := cfg.db.GetChirps()
	respondWithJson(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) getChirpByIdHandler(w http.ResponseWriter, r *http.Request){
	id := r.PathValue("chirpID")
	chirps, _ := cfg.db.GetChirps()

	for _,chirp := range chirps{
		num, _ := strconv.Atoi(id)
		if chirp.Id == num {
			respondWithJson(w, http.StatusOK, chirp)
			return
		}
	}

	respondWithError(w, http.StatusNotFound, "")
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request){

	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	type response struct {
		Id int `json:"id"`
		Email string `json:"email"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&params)

	log.Printf("Email is %s", params.Email)
	user,_ := cfg.db.CreateUser(params.Email, params.Password)

	res := response{
		Id: user.Id,
		Email: user.Email,
	}
	respondWithJson(w, http.StatusCreated, res)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request){
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
		Expires int `json:"expires_in_seconds"`
	}

	type response struct {
		Id int `json:"id"`
		Email string `json:"email"`
		Token string `json:"token"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&params)

	
	user, err := cfg.db.GetUser(params.Email)
	if err!= nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password))

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	fmt.Printf("Secret is %s\n",jwtSecret)

	if params.Expires == 0 {
		params.Expires = 1800
	}
	expi := time.Second * time.Duration(params.Expires)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "chirpy",
		IssuedAt: jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expi)),
		Subject: strconv.Itoa(user.Id),
	})

	log.Printf("token acquired %s", token.Raw)
	signedToken,err := token.SignedString([]byte(jwtSecret))
	if err!= nil {
		log.Printf("Error signing in token")
		return 
	}

	log.Printf("Signin completed")

	res := response{
		Id: user.Id,
		Email: user.Email,
		Token: signedToken,
	}
	respondWithJson(w, http.StatusOK, res)
}

//validate the token befor updating
func (cfg *apiConfig) updateUserHandler(w http.ResponseWriter, r *http.Request){
	bearerToken := r.Header.Get("Authorization")
	token := strings.TrimSpace(strings.TrimPrefix(bearerToken, "Bearer "))

	claims := &jwt.RegisteredClaims{}
	jwtSecret := os.Getenv("JWT_SECRET")

	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
        return []byte(jwtSecret), nil
    })

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	sb, err := parsedToken.Claims.GetSubject()
	fmt.Printf("%s\n", sb)

	if err!= nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	num, _ := strconv.Atoi(sb)

	if err != nil {
		respondWithError(w, 404, "")
		return
	}

	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&params)

	user, _:= cfg.db.PutUserById(num, params.Email, params.Password)

	type response struct {
		Id int `json:"id"`
		Email string `json:"email"`
	}

	res := response{
		Id: user.Id,
		Email: user.Email,
	}
	respondWithJson(w, http.StatusOK, res)
}

func main(){
	log.Printf("Starting the server")
	const port = "8080"
	const filePathRoot = "."

	godotenv.Load()

	mux := http.NewServeMux()
	db,err := database.NewDB("database.json")
	if err != nil{
		log.Fatal(err)
	}

	apicfg := apiConfig{
		fileserverHits: 0,
		db: db,
	}

	mux.Handle("/app/", http.StripPrefix("/app", apicfg.middlewareMetricsInc(http.FileServer(http.Dir(filePathRoot)))))
	mux.Handle("GET /api/healthz", http.HandlerFunc(myCustomHandler))
	mux.Handle("GET /admin/metrics", http.HandlerFunc(apicfg.fileServerHitsLoggerHandler))
	mux.Handle("GET /api/reset", http.HandlerFunc(apicfg.fileServerHitsResteHandler))
	mux.Handle("POST /api/chirps", http.HandlerFunc(apicfg.validateChirpHandler))
	mux.Handle("GET /api/chirps", http.HandlerFunc(apicfg.getChirpHandler))
	mux.Handle("GET /api/chirps/{chirpID}", http.HandlerFunc(apicfg.getChirpByIdHandler))
	mux.Handle("POST /api/users", http.HandlerFunc(apicfg.createUserHandler))
	mux.Handle("POST /api/login", http.HandlerFunc(apicfg.loginHandler))
	mux.Handle("PUT /api/users", http.HandlerFunc(apicfg.updateUserHandler))

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