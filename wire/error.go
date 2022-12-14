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

// UnknownSenderIDError is returned by an attempt to dispatch an
// incoming message that indicates a method call on an object that the
// State doesn't know about.
type UnknownSenderIDError struct {
	Msg *MessageBuffer
}

func (err UnknownSenderIDError) Error() string {
	return fmt.Sprintf("unknown sender object ID: %v", err.Msg.Sender())
}
