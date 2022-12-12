package cursor

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrBadMagic indicates an unrecognized magic number when attempting
// to load a cursor.
var ErrBadMagic = errors.New("bad magic")

const (
	fileMagic = 0x72756358 // ASCII "Xcur"
)

type decoder struct {
	r    io.Reader
	br   *bufio.Reader
	n    int
	err  error
	size int
}

func DecodeFile(path string, size int) (*Cursor, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer file.Close()

	return Decode(file, size)
}

func Decode(r io.Reader, size int) (*Cursor, error) {
	d := decoder{
		r:    r,
		br:   bufio.NewReader(r),
		size: size,
	}
	return d.Decode()
}

func (d *decoder) Decode() (c *Cursor, err error) {
	if d.err != nil {
		return nil, d.err
	}

	defer d.catch(&err)

	tocs := d.header()
	for _, toc := range tocs {
		d.SeekTo(int(toc.Position))
	}

	panic("Not implemented.")
}

func (d *decoder) header() []fileToc {
	magic := d.uint32()
	if magic != fileMagic {
		d.throw(ErrBadMagic)
	}
	hsize := d.uint32()
	d.uint32() // Version.
	ntoc := int(d.uint32())
	d.SeekTo(int(hsize))

	tocs := make([]fileToc, 0, ntoc)
	for i := 0; i < ntoc; i++ {
		tocs = append(tocs, fileToc{
			Type:     d.uint32(),
			Subtype:  d.uint32(),
			Position: d.uint32(),
		})
	}

	return tocs
}

func (d *decoder) uint32() (v uint32) {
	d.throw(binary.Read(d, binary.LittleEndian, &v))
	return v
}

func (d *decoder) Read(buf []byte) (int, error) {
	n, err := d.br.Read(buf)
	d.throw(err)
	d.n += n
	return n, err
}

func (d *decoder) Discard(n int) (int, error) {
	disc, err := d.br.Discard(n)
	d.throw(err)
	d.n += disc
	return disc, err
}

func (d *decoder) SeekTo(n int) error {
	diff := n - d.n
	if diff < 0 {
		panic("tried to seek backwards")
	}
	if diff == 0 {
		return nil
	}

	s, ok := d.r.(io.Seeker)
	if !ok || (diff <= d.br.Buffered()) {
		_, err := d.Discard(diff)
		d.throw(err)
		return nil
	}

	_, err := s.Seek(int64(n), io.SeekStart)
	d.throw(err)
	d.br.Reset(d.r)
	d.n = n
	return nil
}

type fileToc struct {
	Type     uint32
	Subtype  uint32
	Position uint32
}

type decoderError struct {
	err error
}

func (d *decoder) throw(err error) {
	if err != nil {
		panic(decoderError{err: err})
	}
}

func (d *decoder) catch(err *error) {
	switch r := recover().(type) {
	case decoderError:
		*err = r.err
		d.err = r.err
	case nil:
		*err = d.err
	default:
		panic(r)
	}
}
