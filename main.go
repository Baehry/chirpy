package main

import (
	"net/http"
	"fmt"
	"os"
)

func main() {
	mux := http.NewServeMux()
	server := http.Server {
		Handler: mux,
		Addr: ":8080",
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}