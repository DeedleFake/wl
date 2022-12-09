package wl

// PointerButton indicates a mouse button.
type PointerButton uint32

// These values were pulled from linux/input-event-codes.h.
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
