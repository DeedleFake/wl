package main

import (
	"log"

	wl "deedles.dev/wl/client"
)

func main() {
	display, err := wl.DialDisplay()
	if err != nil {
		log.Fatalf("dial display: %v", err)
	}
	defer display.Close()
	display.Error = func(id, code uint32, msg string) {
		log.Printf("error: id: %v, code: %v, msg: %q", id, code, msg)
	}

	registry := display.GetRegistry()

	err = display.RoundTrip()
	if err != nil {
		log.Fatalf("round trip: %v", err)
	}

	for name, inter := range registry.Globals() {
		log.Printf("global: name: %v, interface: %q, version: %v", name, inter.Name, inter.Version)
	}
}
