package wl

// Then is a convenience function that sets c's Listener to an
// implementation that will call f when the Done event is triggered.
func (c *Callback) Then(f func(uint32)) {
	c.Listener = callbackListener(f)
}

type callbackListener func(uint32)

func (lis callbackListener) Done(data uint32) {
	lis(data)
}
