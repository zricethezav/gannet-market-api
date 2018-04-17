package main

import (
	"net/http"
	"log"
	"fmt"
	"regexp"
	"sync"
	"encoding/json"
)

type Produce struct {
	Code  string
	Name string
	Price float32 `json:",string"`
}

type produceDB struct {
	data []*Produce
	lock sync.Mutex
}

// globals include the database, cache, and regex
var (
	db produceDB
	dbCache map[string]bool
	nameRegexp *regexp.Regexp
	codeRegexp *regexp.Regexp
)

func init() {
	nameRegexp = regexp.MustCompile("[0-9A-Za-z]$")
	codeRegexp = regexp.MustCompile("[0-9A-Za-z]{4}-[0-9A-Za-z]{4}-[0-9A-Za-z]{4}-[0-9A-Za-z]{4}$")
	db = produceDB{}
	dbCache = map[string]bool{}
}

func main() {
	log.Println("Starting gannet-market-api service")
	http.HandleFunc("/", invalidHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/fetch", fetchHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func invalidHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "page not found", http.StatusNotFound)
}

// addHandler()
func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fmt.Fprint(w, "/add accepts POST requests")
		return
	}
	var p Produce
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, "unable to proces request", http.StatusUnprocessableEntity)
		return
	}
	if !(nameRegexp.Match([]byte(p.Name))) {
		http.Error(w, "invalid name", http.StatusUnprocessableEntity)
		return
	}
	if !(codeRegexp.Match([]byte(p.Code))) {
		http.Error(w, "invalid code", http.StatusUnprocessableEntity)
		return
	}
	err = db.add(&p)
	if err != nil {
		http.Error(w, "entry already exists", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// deleteHandler()
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if !(codeRegexp.Match([]byte(code))) {
		http.Error(w, "entry does not exist", http.StatusUnprocessableEntity)
		return
	}
	err := db.delete(code)
	if err != nil {
		http.Error(w, "entry does not exists", http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// fetchHandler()
func fetchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		fmt.Fprint(w, "/fetch accepts GET requests")
		return
	}
	resp, err := json.Marshal(db.data)
	if err != nil {
		http.Error(w, "failed to create entry", http.StatusForbidden)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

// ProduceDB add() is responsible for adding a produce entry to the produce database.
// If the produce code already exists in the database, a 409 will be returned. Writes
// protected by a mutex to ensure only one writer.
func (db *produceDB) add(produce *Produce) error {
	if _, exists := dbCache[produce.Name]; exists {
		return fmt.Errorf("code already exists")
	}

	db.lock.Lock()
	defer db.lock.Unlock()
	// update database and cache
	db.data = append(db.data, produce)
	dbCache[produce.Name] = true
	return nil
}

// ProduceDB delete() is responsible for removing produce from the produce database
// based on a produce code.
func (db *produceDB) delete(code string) error {
	for i, produce := range db.data {
		if produce.Code == code {
			db.lock.Lock()
			defer db.lock.Unlock()
			// remove from db, update cache
			copy(db.data[i:], db.data[i+1:])
			db.data[len(db.data)-1] = nil
			db.data = db.data[:len(db.data)-1]
			dbCache[code] = false
			return nil
		}
	}
	return fmt.Errorf("entry does not exist")
}

// helper scripts
// curl -H "Content-Type: application/json" -X POST -d '{"name":"apple","code":"YRT6-72AS-K736-L4AR", "price": "12.12"}' localhost:8080
