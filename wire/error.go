package wire

import (
	"fmt"
)

// UnknownOpError is returned by Object.Dispatch if it is given a
// message with an invalid opcode.
type UnknownOpError struct {
	Interface string
	Type      string
	Op        uint16
}

func (err UnknownOpError) Error() string {
	return fmt.Sprintf("unknown %v opcode for %v: %v", err.Type, err.Interface, err.Op)
}
