package wl

type Surface struct {
	id[surfaceObject]
	display *Display
}

func (s *Surface) Attach(buf *Buffer, x, y int32) {
	s.display.Enqueue(s.obj.Attach(buf.obj.id, x, y))
}

func (s *Surface) Damage(x, y, width, height int32) {
	s.display.Enqueue(s.obj.Damage(x, y, width, height))
}

func (s *Surface) Commit() {
	s.display.Enqueue(s.obj.Commit())
}

type surfaceListener struct {
	surface *Surface
}

func (lis surfaceListener) Enter(output uint32) {
	// TODO
}

func (lis surfaceListener) Leave(output uint32) {
	// TODO
}
