package wl

import (
	"os"

	"deedles.dev/wl/wire"
)

type Keyboard struct {
	Keymap     func(format KeyboardKeymapFormat, file *os.File, size uint32)
	Enter      func(serial uint32, surface *Surface, keys []byte)
	Leave      func(serial uint32, surface *Surface)
	Key        func(serial, time, key uint32, state KeyboardKeyState)
	Modifiers  func(serial, depressed, latched, locked, group uint32)
	RepeatInfo func(rate, delay int32)

	obj     keyboardObject
	display *Display
}

func (kb *Keyboard) Object() wire.Object {
	return &kb.obj
}

type keyboardListener struct {
	kb *Keyboard
}

func (lis keyboardListener) Keymap(format uint32, fd *os.File, size uint32) {
	if lis.kb.Keymap != nil {
		lis.kb.Keymap(KeyboardKeymapFormat(format), fd, size)
		return
	}
	fd.Close()
}

func (lis keyboardListener) Enter(serial uint32, surface uint32, keys []byte) {
	if lis.kb.Enter != nil {
		var s *Surface
		if sobj, ok := lis.kb.display.GetObject(surface).(*Surface); ok {
			s = sobj
		}
		lis.kb.Enter(serial, s, keys)
	}
}

func (lis keyboardListener) Leave(serial uint32, surface uint32) {
	if lis.kb.Leave != nil {
		var s *Surface
		if sobj, ok := lis.kb.display.GetObject(surface).(*Surface); ok {
			s = sobj
		}
		lis.kb.Leave(serial, s)
	}
}

func (lis keyboardListener) Key(serial uint32, time uint32, key uint32, state uint32) {
	if lis.kb.Key != nil {
		lis.kb.Key(serial, time, key, KeyboardKeyState(state))
	}
}

func (lis keyboardListener) Modifiers(serial uint32, modsDepressed uint32, modsLatched uint32, modsLocked uint32, group uint32) {
	if lis.kb.Modifiers != nil {
		lis.kb.Modifiers(serial, modsDepressed, modsLatched, modsLocked, group)
	}
}

func (lis keyboardListener) RepeatInfo(rate int32, delay int32) {
	if lis.kb.RepeatInfo != nil {
		lis.kb.RepeatInfo(rate, delay)
	}
}
