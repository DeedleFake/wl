package wl

import "deedles.dev/wl/wire"

type Pointer struct {
	Enter        func(serial uint32, s *Surface, x, y wire.Fixed)
	Leave        func(serial uint32, s *Surface)
	Motion       func(time uint32, x, y wire.Fixed)
	Button       func(serial, time uint32, button PointerButton, state PointerButtonState)
	Axis         func(time uint32, axis PointerAxis, value wire.Fixed)
	Frame        func()
	AxisSource   func(PointerAxisSource)
	AxisStop     func(time uint32, axis PointerAxis)
	AxisDiscrete func(axis PointerAxis, discrete int32)

	obj     pointerObject
	display *Display
}

func (p *Pointer) Object() wire.Object {
	return &p.obj
}

type pointerListener struct {
	p *Pointer
}

func (lis pointerListener) Enter(serial uint32, surface uint32, surfaceX wire.Fixed, surfaceY wire.Fixed) {
	if lis.p.Enter != nil {
		var s *Surface
		if so, ok := lis.p.display.GetObject(surface).(*Surface); ok {
			s = so
		}
		lis.p.Enter(serial, s, surfaceX, surfaceY)
	}
}

func (lis pointerListener) Leave(serial uint32, surface uint32) {
	if lis.p.Leave != nil {
		var s *Surface
		if so, ok := lis.p.display.GetObject(surface).(*Surface); ok {
			s = so
		}
		lis.p.Leave(serial, s)
	}
}

func (lis pointerListener) Motion(time uint32, surfaceX wire.Fixed, surfaceY wire.Fixed) {
	if lis.p.Motion != nil {
		lis.p.Motion(time, surfaceX, surfaceY)
	}
}

func (lis pointerListener) Button(serial uint32, time uint32, button uint32, state uint32) {
	if lis.p.Button != nil {
		lis.p.Button(serial, time, PointerButton(button), PointerButtonState(state))
	}
}

func (lis pointerListener) Axis(time uint32, axis uint32, value wire.Fixed) {
	if lis.p.Axis != nil {
		lis.p.Axis(time, PointerAxis(axis), value)
	}
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
	if lis.p.AxisStop != nil {
		lis.p.AxisStop(time, PointerAxis(axis))
	}
}

func (lis pointerListener) AxisDiscrete(axis uint32, discrete int32) {
	if lis.p.AxisDiscrete != nil {
		lis.p.AxisDiscrete(PointerAxis(axis), discrete)
	}
}

type PointerButton uint32

const (
	PointerButtonLeft PointerButton = 0x110 + iota
	PointerButtonRight
	PointerButtonMiddle
	PointerButtonSide
	PointerButtonExtra
	PointerButtonForward
	PointerButtonBack
	PointerButtonTask
)

func (b PointerButton) String() string {
	switch b {
	case PointerButtonLeft:
		return "left"
	case PointerButtonRight:
		return "right"
	case PointerButtonMiddle:
		return "middle"
	case PointerButtonSide:
		return "side"
	case PointerButtonExtra:
		return "extra"
	case PointerButtonForward:
		return "forward"
	case PointerButtonBack:
		return "back"
	case PointerButtonTask:
		return "task"
	}

	return "unknown"
}
