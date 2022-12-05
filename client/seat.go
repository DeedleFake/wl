package wl

type Seat struct {
	Capabilities func(SeatCapability)
	Name         func(string)

	id[seatObject]
	display *Display
}

func IsSeat(i Interface) bool {
	return i.Is(seatInterface, seatVersion)
}

func BindSeat(display *Display, name uint32) *Seat {
	seat := Seat{display: display}
	seat.obj.listener = seatListener{seat: &seat}
	display.AddObject(&seat.obj)

	registry := display.GetRegistry()
	registry.Bind(name, seatInterface, seatVersion, seat.obj.id)

	return &seat
}

func (seat *Seat) Release() {
	seat.display.Enqueue(seat.obj.Release())
	seat.display.DeleteObject(seat.obj.id)
}

func (seat *Seat) GetKeyboard() *Keyboard {
	keyboard := Keyboard{display: seat.display}
	keyboard.obj.listener = keyboardListener{kb: &keyboard}
	seat.display.AddObject(&keyboard.obj)
	seat.display.Enqueue(seat.obj.GetKeyboard(keyboard.obj.id))
	return &keyboard
}

func (seat *Seat) GetPointer() *Pointer {
	pointer := Pointer{display: seat.display}
	pointer.obj.listener = pointerListener{p: &pointer}
	seat.display.AddObject(&pointer.obj)
	seat.display.Enqueue(seat.obj.GetPointer(pointer.obj.id))
	return &pointer
}

type seatListener struct {
	seat *Seat
}

func (lis seatListener) Capabilities(cap uint32) {
	if lis.seat.Capabilities != nil {
		lis.seat.Capabilities(SeatCapability(cap))
	}
}

func (lis seatListener) Name(name string) {
	if lis.seat.Name != nil {
		lis.seat.Name(name)
	}
}
