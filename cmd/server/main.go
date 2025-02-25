package main

import "net/http"

// run server successfully or return error to panic in the main()
func run() error {
	mux := http.NewServeMux()
	err := http.ListenAndServe("localhost:8080", mux)
	return err
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}
