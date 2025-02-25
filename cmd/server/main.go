package main

import (
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/handlers"
)

// run server successfully or return error to panic in the main()
func run() error {
	mux := http.NewServeMux()
	mux.HandleFunc(`/update/`, handlers.Update)
	err := http.ListenAndServe("localhost:8080", mux)
	return err
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}
