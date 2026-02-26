package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("application/json", "Content-Type")
		w.Write([]byte(`{"Status": "ok"}`))
	})

	mux.HandleFunc("POST /echo", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&payload)
		if err != nil {
			fmt.Println(err)
			return
		}
		json.NewEncoder(w).Encode(payload)
	})
	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	server.ListenAndServe()
}
