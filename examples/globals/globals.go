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

	var outputs []*wl.Output

	registry := display.GetRegistry()
	registry.Global = func(name uint32, inter wl.Interface) {
		log.Printf("global: name: %v, interface: %q, version: %v", name, inter.Name, inter.Version)

		switch {
		case wl.IsOutput(inter):
			outputs = append(outputs, wl.BindOutput(display, name))
		}
	}

	err = display.RoundTrip()
	if err != nil {
		log.Fatalf("round trip: %v", err)
	}

	for _, out := range outputs {
		out.Geometry = func(x, y, w, h, sp int32, make, model string, transform wl.OutputTransform) {
			log.Printf(
				"output: x: %v, y: %v, w: %v, h: %v, subpixel: %v, make: %q, model: %q, transform: %v",
				x,
				y,
				w,
				h,
				sp,
				make,
				model,
				transform,
			)
		}
	}

	err = display.RoundTrip()
	if err != nil {
		log.Fatalf("round trip: %v", err)
	}
}
