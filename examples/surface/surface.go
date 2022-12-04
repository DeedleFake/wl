package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	wl "deedles.dev/wl/client"
)

func main() {
	display, err := wl.DialDisplay()
	if err != nil {
		log.Fatalf("dial display: %v", err)
	}
	defer display.Close()

	registry := display.GetRegistry()

	var (
		compositor *wl.Compositor
		shm        *wl.Shm
	)
	registry.Global = func(name uint32, inter wl.Interface) {
		switch {
		case wl.IsCompositor(inter):
			compositor = wl.BindCompositor(display, name)
		case wl.IsShm(inter):
			shm = wl.BindShm(display, name)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err = display.RoundTrip()
		if err != nil {
			log.Fatalf("round trip: %v", err)
		}

		fmt.Println(compositor)
		fmt.Println(shm)
	}
}
