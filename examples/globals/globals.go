package main

import (
	"log"

	wl "deedles.dev/wl/client"
	"deedles.dev/wl/wire"
)

func main() {
	c, err := wire.Dial()
	if err != nil {
		log.Fatalf("dial socket: %v", err)
	}
	display := wl.ConnectDisplay(c)
	defer display.Close()

	registry := display.GetRegistry()
	registry.Global = func(name uint32, inter string, version uint32) {
		log.Printf("global: name: %v, interface: %q, version: %v", name, inter, version)
	}

	err = display.RoundTrip()
	if err != nil {
		log.Fatalf("round trip: %v", err)
	}
}
