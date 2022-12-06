package wl

import "deedles.dev/wl/wire"

type callback struct {
	Done func(data uint32)

	obj callbackObject
}

func (cb *callback) Object() wire.Object {
	return &cb.obj
}

type callbackListener struct {
	callback *callback
}

func (lis callbackListener) Done(data uint32) {
	if lis.callback.Done != nil {
		lis.callback.Done(data)
	}
}
