package main

import (
	"log"

	wl "deedles.dev/wl/server"
)

func main() {
	server, err := wl.ListenAndServe()
	if err != nil {
		log.Fatalf("start server: %v", err)
	}
	defer server.Close()

	panic("Not implemented.")
}
