package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main3() {
	mux := http.NewServeMux()

	type User struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	}

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		u := User{Id: 20, Name: "Alex"}
		if err := json.NewEncoder(w).Encode(u); err != nil {
			fmt.Println("Failed", http.StatusInternalServerError)
		}
		return
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Println("Starting server at: %s", srv.Addr)
	err := srv.ListenAndServe()
	if err != nil {
		fmt.Println("Failed to start server: %s", srv.Addr)
	}
}
