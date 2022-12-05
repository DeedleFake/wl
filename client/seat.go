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
