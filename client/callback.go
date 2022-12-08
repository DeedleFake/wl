package wl

func (c *Callback) Then(f func(uint32)) {
	c.Listener = callbackListener(f)
}

type callbackListener func(uint32)

func (lis callbackListener) Done(data uint32) {
	lis(data)
}
