package wire

import (
	"fmt"
	"math"
	"strings"
	"unsafe"
)

// Fixed is a 24_8 fixed-point number. Wayland does not have support
// for floating point numbers in its core protocol and uses these
// instead.
type Fixed int32

func FixedInt(v int) Fixed {
	return Fixed(v << 8)
}

func FixedFloat(v float64) Fixed {
	i, frac := math.Modf(v)
	return Fixed((int(i) << 8) | int(math.Abs(frac)*math.Exp2(8)))
}

func (f Fixed) Int() int {
	return int(f >> 8)
}

func (f Fixed) Frac() int {
	return int(*(*uint32)(unsafe.Pointer(&f)) & 0xFF)
}

func (f Fixed) Float() float64 {
	i := f.Int()
	frac := f.Frac()
	return float64(i) + math.Abs(float64(frac)*math.Exp2(-8))
}

func (f Fixed) String() string {
	var sb strings.Builder
	fmt.Fprint(&sb, f.Int())
	if frac := f.Frac(); frac != 0 {
		fmt.Fprintf(&sb, ".%v", frac)
	}
	return sb.String()
}
