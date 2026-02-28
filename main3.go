package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {

	mux := http.NewServeMux()

	type User struct {
		Name string `json:"name"`
	}

	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			fmt.Println("Это не GET", http.StatusInternalServerError)
		}
		fmt.Println("Helloworld")
		data := map[string]string{
			"message": "saval",
			"time":    time.Now().String(),
		}
		json.NewEncoder(w).Encode(data)

	})

	mux.HandleFunc("GET /user", func(w http.ResponseWriter, r *http.Request) {
		u := User{"masha"}
		v := User{"sago"}
		json.NewEncoder(w).Encode(u)
		json.NewEncoder(w).Encode(v)
		w.Header().Set("Content-Type", "applicaton/json")

	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal("Pizda", err)
	}
}
