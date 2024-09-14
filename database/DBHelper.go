package database

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

type DB struct {
	path string
	mu *sync.RWMutex
}

type Chirp struct {
	Id int `json:"id"`
	Body string `json:"body"`
}

type DBStructure struct {
	Chirps map[int] Chirp `json:"chirps"`
}

func (db *DB) CreateChirp(body string) (Chirp, error){
	dbStructure, err := db.loadDB()
	if err!= nil{
		return Chirp{}, err
	}
	
	id := len(dbStructure.Chirps)+1

	chirp := Chirp{
		Id: id,
		Body: body,
	}

	dbStructure.Chirps[id]=chirp
	err = db.writeDB(dbStructure)
	if err != nil{
		return Chirp{}, err
	}
	return chirp, nil
}

func (db *DB) GetChirps()([]Chirp, error) {
	dbStructure, err := db.loadDB()

	if err != nil {
		return nil, err
	}

	chirps := make([]Chirp, 0, len(dbStructure.Chirps))

	for _,chirp := range dbStructure.Chirps {
		chirps = append(chirps, chirp)
	}
	return chirps, nil
}

func (db *DB) loadDB() (DBStructure, error){

	db.mu.Lock()
	defer db.mu.Unlock()

	data, err := os.ReadFile(db.path)
	if err != nil {
		return DBStructure{}, err
	}

	dbStructure := DBStructure{}

	err = json.Unmarshal(data, &dbStructure)
	if err != nil {
		return dbStructure, err
	}
	return dbStructure, nil
}

func NewDB(path string) (*DB, error){
	db := &DB{
		path: path,
		mu: &sync.RWMutex{},
	}

	err := db.ensureDB()
	return db, err
}

func (db *DB) ensureDB() error {
	_,err := os.ReadFile(db.path)

	if errors.Is(err, os.ErrNotExist) {
		return db.createDB()
	}
	return err
}

func (db *DB) createDB() error {
	dbStructure := DBStructure{
		Chirps: map[int]Chirp{},
	}
	return db.writeDB(dbStructure)
}

func (db *DB) writeDB(dbStructure DBStructure) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, err := json.Marshal(dbStructure)
	if err != nil {
		return err
	}

	err = os.WriteFile(db.path, data, 0600)
	return err
}