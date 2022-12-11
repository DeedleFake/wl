package shm

import (
	"fmt"
	"image"
	"image/draw"
	"os"

	wl "deedles.dev/wl/client"
	"deedles.dev/wl/shm/shmimage"
	"golang.org/x/sys/unix"
)

type ImageBuffer struct {
	w, h int32
	shm  *wl.Shm
	pool *wl.ShmPool
	buf  *wl.Buffer
	file *os.File
	mmap Mmap
}

func NewImageBuffer(shm *wl.Shm, w, h int32) (s *ImageBuffer, err error) {
	defer func() {
		if err != nil {
			s.Destroy()
		}
	}()

	s = &ImageBuffer{
		w:   w,
		h:   h,
		shm: shm,
	}
	cap := s.Stride() * s.h

	file, err := Create()
	if err != nil {
		return s, fmt.Errorf("create SHM file: %w", err)
	}
	s.file = file
	s.file.Truncate(int64(cap))

	mmap, err := MapShared(file, int(cap), unix.PROT_READ|unix.PROT_WRITE)
	if err != nil {
		return s, fmt.Errorf("mmap SHM file: %w", err)
	}
	s.mmap = mmap

	s.pool = s.shm.CreatePool(file, int32(len(s.mmap)))
	s.buf = s.pool.CreateBuffer(0, w, h, w*4, wl.ShmFormatArgb8888)

	return s, nil
}

func (s *ImageBuffer) Destroy() {
	s.mmap.Unmap()
	s.file.Close()
	s.buf.Destroy()
	s.pool.Destroy()
}

func (s *ImageBuffer) Shm() *wl.Shm {
	return s.shm
}

func (s *ImageBuffer) ShmPool() *wl.ShmPool {
	return s.pool
}

func (s *ImageBuffer) Buffer() *wl.Buffer {
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
		s.buf = s.pool.CreateBuffer(0, s.w, s.h, s.Stride(), wl.ShmFormatArgb8888)
		return nil
	}

	s.file.Truncate(int64(s.Len()))

	err := s.mmap.Unmap()
	if err != nil {
		return fmt.Errorf("unmap: %w", err)
	}
	mmap, err := MapShared(s.file, int(s.Len()), unix.PROT_READ|unix.PROT_WRITE)
	if err != nil {
		return fmt.Errorf("mmap: %w", err)
	}
	s.mmap = mmap

	s.buf.Destroy()
	s.pool.Resize(s.Len())
	s.buf = s.pool.CreateBuffer(0, s.w, s.h, s.Stride(), wl.ShmFormatArgb8888)

	return nil
}

func (s *ImageBuffer) Image() draw.Image {
	return &shmimage.ARGB8888{
		Pix:    s.mmap,
		Stride: int(s.Stride()),
		Rect:   s.Bounds(),
	}
}
