package main

import (
	"log"

	wl "deedles.dev/wl/client"
)

type listener struct {
	state    *wl.Client
	registry *wl.Registry
}

type displayListener listener

func (lis *displayListener) Error(id, code uint32, msg string) {
	log.Printf("error: id: %v, code: %v, msg: %q", id, code, msg)
}

func (lis *displayListener) DeleteId(id uint32) {
	lis.state.Delete(id)
}

type registryListener listener

func (lis *registryListener) Global(name uint32, inter string, version uint32) {
	log.Printf("global: name: %v, interface: %q, version: %v", name, inter, version)

	switch inter {
	case wl.OutputInterface:
		output := wl.BindOutput(lis.state, lis.registry, name, version)
		output.Listener = (*outputListener)(lis)
	}
}

func (lis *registryListener) GlobalRemove(name uint32) {}

type outputListener listener

func (lis *outputListener) Geometry(x, y, w, h int32, subpixel wl.OutputSubpixel, make, model string, transform wl.OutputTransform) {
	log.Printf(
		"output: x: %v, y: %v, w: %v, h: %v, subpixel: %v, make: %q, model: %q, transform: %v",
		x,
		y,
		w,
		h,
		subpixel,
		make,
		model,
		transform,
	)
}

func (lis *outputListener) Mode(flags wl.OutputMode, w, h, refresh int32) {}

func (lis *outputListener) Done() {}

func (lis *outputListener) Scale(factor int32) {}

func main() {
	s, err := wl.Dial()
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer s.Close()

	display := s.Display()
	registry := display.GetRegistry()

	lis := listener{
		state:    s,
		registry: registry,
	}
	display.Listener = (*displayListener)(&lis)
	registry.Listener = (*registryListener)(&lis)

	err = s.RoundTrip()
	if err != nil {
		log.Fatalf("round trip: %v", err)
	}

	err = s.RoundTrip()
	if err != nil {
		log.Fatalf("round trip: %v", err)
	}
}
