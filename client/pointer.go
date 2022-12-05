package wl

import "deedles.dev/wl/wire"

type Pointer struct {
	Frame      func()
	AxisSource func(PointerAxisSource)

	id[pointerObject]
	display *Display
}

type pointerListener struct {
	p *Pointer
}

func (lis pointerListener) Enter(serial uint32, surface uint32, surfaceX wire.Fixed, surfaceY wire.Fixed) {
	// TODO
}

func (lis pointerListener) Leave(serial uint32, surface uint32) {
	// TODO
}

func (lis pointerListener) Motion(time uint32, surfaceX wire.Fixed, surfaceY wire.Fixed) {
	// TODO
}

func (lis pointerListener) Button(serial uint32, time uint32, button uint32, state uint32) {
	// TODO
}

func (lis pointerListener) Axis(time uint32, axis uint32, value wire.Fixed) {
	// TODO
}

func (lis pointerListener) Frame() {
	if lis.p.Frame != nil {
		lis.p.Frame()
	}
}

func (lis pointerListener) AxisSource(axisSource uint32) {
	if lis.p.AxisSource != nil {
		lis.p.AxisSource(PointerAxisSource(axisSource))
	}
}

func (lis pointerListener) AxisStop(time uint32, axis uint32) {
	// TODO
}

func (lis pointerListener) AxisDiscrete(axis uint32, discrete int32) {
	// TODO
}
