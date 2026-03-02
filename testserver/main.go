package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Args[1]

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend on port %s — path: %s\n", port, r.URL.Path)
	})

	http.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second)
		fmt.Fprintf(w, "too late\n")
	})

	fmt.Println("Test server running on port", port)
	http.ListenAndServe(":"+port, nil)
}