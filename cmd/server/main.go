package main

import (
	"github.com/PiskarevSA/go-advanced/internal/app/server"
)

func main() {
	server := server.NewServer()
	err := server.Run()
	if err != nil {
		panic(err)
	}
}
