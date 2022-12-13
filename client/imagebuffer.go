package wl

import (
	"fmt"
	"image"
	"image/draw"
	"os"

	"deedles.dev/wl/shm"
	"deedles.dev/ximage"
	"golang.org/x/sys/unix"
)

type ImageBuffer struct {
	w, h int32
	shm  *Shm
	pool *ShmPool
	buf  *Buffer
	file *os.File
	mmap shm.Mmap
}

func NewImageBuffer(s *Shm, w, h int32) (buf *ImageBuffer, err error) {
	defer func() {
		if err != nil {
			buf.Destroy()
		}
	}()

	buf = &ImageBuffer{
		w:   w,
		h:   h,
		shm: s,
	}
	cap := buf.Stride() * buf.h

	file, err := shm.Create()
	if err != nil {
		return buf, fmt.Errorf("create SHM file: %w", err)
	}
	buf.file = file
	buf.file.Truncate(int64(cap))

	mmap, err := shm.MapShared(file, int(cap), unix.PROT_READ|unix.PROT_WRITE)
	if err != nil {
		return buf, fmt.Errorf("mmap SHM file: %w", err)
	}
	buf.mmap = mmap

	buf.pool = buf.shm.CreatePool(file, int32(len(buf.mmap)))
	buf.buf = buf.pool.CreateBuffer(0, w, h, w*4, ShmFormatArgb8888)

	return buf, nil
}

func (s *ImageBuffer) Destroy() {
	if s.mmap != nil {
		s.mmap.Unmap()
	}
	if s.file != nil {
		s.file.Close()
	}
	if s.buf != nil {
		s.buf.Destroy()
	}
	if s.pool != nil {
		s.pool.Destroy()
	}
}

func (s *ImageBuffer) Shm() *Shm {
	return s.shm
}

func (s *ImageBuffer) ShmPool() *ShmPool {
	return s.pool
}

func (s *ImageBuffer) Buffer() *Buffer {
	return s.buf
}

func (s *ImageBuffer) Stride() int32 {
	return s.w * 4
}

func (s *ImageBuffer) Len() int32 {
	return s.Stride() * s.h
}

func (s *ImageBuffer) Cap() int32 {
	return int32(cap(s.mmap))
}

func (s *ImageBuffer) Bounds() image.Rectangle {
	return image.Rect(
		0,
		0,
		int(s.w),
		int(s.h),
	)
}

func (s *ImageBuffer) Resize(w, h int32) error {
	if (w == s.w) && (h == s.h) {
		return nil
	}

	s.w = w
	s.h = h
	if s.Len() < s.Cap() {
		s.mmap = s.mmap[:s.Len()]
		s.buf.Destroy()
		s.buf = s.pool.CreateBuffer(0, s.w, s.h, s.Stride(), ShmFormatArgb8888)
		return nil
	}

	s.file.Truncate(int64(s.Len()))

	err := s.mmap.Unmap()
	if err != nil {
		return fmt.Errorf("unmap: %w", err)
	}
	mmap, err := shm.MapShared(s.file, int(s.Len()), unix.PROT_READ|unix.PROT_WRITE)
	if err != nil {
		return fmt.Errorf("mmap: %w", err)
	}
	s.mmap = mmap

	s.buf.Destroy()
	s.pool.Resize(s.Len())
	s.buf = s.pool.CreateBuffer(0, s.w, s.h, s.Stride(), ShmFormatArgb8888)

	return nil
}

func (s *ImageBuffer) Image() draw.Image {
	return &ximage.FormatImage{
		Format: ximage.ARGB8888,
		Rect:   s.Bounds(),
		Pix:    s.mmap,
	}
}
