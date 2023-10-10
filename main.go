package main

import (
	"fmt"
	"net/http"

	_ "github.com/go-chi/chi/v5"
)

func main() {
	server := &http.Server{
		Addr:    ":3000",
		Handler: http.HandlerFunc(basicHandler),
	}

	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("failed to listen to server", err)
	}
}

func basicHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if r.URL.Path == "/sayHello" {
			w.Write([]byte("Hello world!"))
		}
	}
}