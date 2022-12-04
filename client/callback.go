package wl

type callback struct {
	Done func(data uint32)

	id[callbackObject]
}

type callbackListener struct {
	callback *callback
}

func (lis callbackListener) Done(data uint32) {
	if lis.callback.Done != nil {
		lis.callback.Done(data)
	}
}
