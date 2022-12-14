// Package pointer contains utilities for handling pointer input.
package pointer

// Button indicates a mouse button.
type Button uint32

// These values were pulled from linux/input-event-codes.h.
const (
	ButtonLeft Button = 0x110 + iota
	ButtonRight
	ButtonMiddle
	ButtonSide
	ButtonExtra
	ButtonForward
	ButtonBack
	ButtonTask
)

func (b Button) String() string {
	switch b {
	case ButtonLeft:
		return "left"
	case ButtonRight:
		return "right"
	case ButtonMiddle:
		return "middle"
	case ButtonSide:
		return "side"
	case ButtonExtra:
		return "extra"
	case ButtonForward:
		return "forward"
	case ButtonBack:
		return "back"
	case ButtonTask:
		return "task"
	}

	return "unknown"
}
