package wl

import "deedles.dev/wl/wire"

type Output struct {
	Geometry func(x, y, physicalWidth, physicalHeight, subpixel int32, make, model string, transform OutputTransform)
	Mode     func(flags OutputMode, width, height, refresh int32)
	Done     func()
	Scale    func(factor int32)

	obj     outputObject
	display *Display
}

func IsOutput(i Interface) bool {
	return i.Is(outputInterface, outputVersion)
}

func BindOutput(display *Display, name uint32) *Output {
	output := Output{display: display}
	output.obj.listener = outputListener{output: &output}
	display.AddObject(&output)

	registry := display.GetRegistry()
	registry.Bind(name, outputInterface, outputVersion, output.obj.id)

	return &output
}

func (out *Output) Object() wire.Object {
	return &out.obj
}

func (out *Output) Release() {
	out.display.Enqueue(out.obj.Release())
	out.display.DeleteObject(out.obj.id)
}

type outputListener struct {
	output *Output
}

func (lis outputListener) Geometry(x, y, physicalWidth, physicalHeight, subpixel int32, make, model string, transform int32) {
	if lis.output.Geometry != nil {
		lis.output.Geometry(x, y, physicalWidth, physicalHeight, subpixel, make, model, OutputTransform(transform))
	}
}

func (lis outputListener) Mode(flags uint32, width, height, refresh int32) {
	if lis.output.Mode != nil {
		lis.output.Mode(OutputMode(flags), width, height, refresh)
	}
}

func (lis outputListener) Done() {
	if lis.output.Done != nil {
		lis.output.Done()
	}
}

func (lis outputListener) Scale(factor int32) {
	if lis.output.Scale != nil {
		lis.output.Scale(factor)
	}
}
