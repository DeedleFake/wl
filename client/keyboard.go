package wl

import "os"

type Keyboard struct {
	Keymap     func(format KeyboardKeymapFormat, file *os.File, size uint32)
	Key        func(serial, time, key uint32, state KeyboardKeyState)
	RepeatInfo func(rate, delay int32)

	id[keyboardObject]
	display *Display
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
	// TODO
}

func (lis keyboardListener) Leave(serial uint32, surface uint32) {
	// TODO
}

func (lis keyboardListener) Key(serial uint32, time uint32, key uint32, state uint32) {
	if lis.kb.Key != nil {
		lis.kb.Key(serial, time, key, KeyboardKeyState(state))
	}
}

func (lis keyboardListener) Modifiers(serial uint32, modsDepressed uint32, modsLatched uint32, modsLocked uint32, group uint32) {
	// TODO
}

func (lis keyboardListener) RepeatInfo(rate int32, delay int32) {
	if lis.kb.RepeatInfo != nil {
		lis.kb.RepeatInfo(rate, delay)
	}
}
