package cursor

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	magic         = 0x72756358
	fileHeaderLen = 4 * 4
)

type ximage struct {
	version uint32
	size    uint32
	width   uint32
	height  uint32
	xhot    uint32
	yhot    uint32
	delay   uint32
	pixels  []byte
}

type ximages struct {
	name   string
	images []ximage
}

func loadTheme(theme string, size int, f func(ximages)) error {
	if theme == "" {
		theme = "default"
	}

	var inherits []string
	for _, path := range libraryPaths() {
		dir := filepath.Join(path, theme)
		if dir == "" {
			continue
		}

		full := filepath.Join(dir, "cursors")
		err := loadAllCursorsFromDir(full, size, f)
		if err != nil {
			return err
		}

		if len(inherits) == 0 {
			full := filepath.Join(dir, "index.theme")
			i, err := themeInherits(full)
			if err == nil {
				inherits = i
			}
		}
	}
	for _, path := range inherits {
		loadTheme(path, size, f)
	}

	return nil
}

var defaultLibraryPaths = []string{
	"~/.icons",
	"/usr/share/icons",
	"/usr/share/pixmaps",
	"~/.cursors",
	"/usr/share/cursors/xorg-x11",
	"/usr/X11R6/lib/X11/icons",
}

func libraryPaths() []string {
	if v, ok := os.LookupEnv("XCURSUR_PATH"); ok {
		return filepath.SplitList(v)
	}

	v, ok := os.LookupEnv("XDG_DATA_HOME")
	if !ok || !filepath.IsAbs(v) {
		v = "~/.local/share"
	}
	return append([]string{filepath.Join(v, "icons")}, defaultLibraryPaths...)
}

func loadAllCursorsFromDir(path string, size int, f func(ximages)) error {
	dir, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	for _, ent := range dir {
		if t := ent.Type().Type(); !t.IsRegular() && (t != fs.ModeSymlink) {
			continue
		}

		loadAllCursorsFromFile(path, ent, size, f)
	}

	return nil
}

func loadAllCursorsFromFile(path string, ent fs.DirEntry, size int, f func(ximages)) error {
	full := filepath.Join(path, ent.Name())
	file, err := os.Open(full)
	if err != nil {
		return err
	}
	defer file.Close()

	images, err := xcFileLoadImages(file, size)
	if err != nil {
		return err
	}

	f(ximages{
		name:   ent.Name(),
		images: images,
	})

	return nil
}

func themeInherits(full string) (inherits []string, err error) {
	if full == "" {
		return nil, nil
	}

	file, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	s := bufio.NewScanner(file)
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "Inherits") {
			continue
		}

		_, after, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		inherits = strings.FieldsFunc(after, func(c rune) bool {
			return (c == ':') || (c == ',')
		})
		for i, v := range inherits {
			inherits[i] = strings.TrimSpace(v)
		}

		break
	}
	if err := s.Err(); err != nil {
		return inherits, fmt.Errorf("scan: %w", err)
	}

	return inherits, nil
}

func xcFileLoadImages(r io.ReadSeeker, size int) ([]ximage, error) {
	hdr, err := readFileHeader(r)
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	bestSize, nsize, err := fileBestSize(hdr, size)
	if err != nil {
		return nil, fmt.Errorf("read best size: %w", err)
	}

	images := make([]ximage, 0, nsize)
	for i := 0; i < nsize; i++ {
		toc, err := findImageToc(hdr, bestSize, i)
		if err != nil {
			return nil, fmt.Errorf("find image toc: %w", err)
		}

		img, err := readImage(r, hdr, toc)
		if err != nil {
			return nil, fmt.Errorf("read image: %w", err)
		}

		images[i] = img
	}

	return images, nil
}

type fileHeader struct {
	Header  uint32
	Version uint32
	Tocs    []fileToc
}

type fileToc struct {
	Type     uint32
	Subtype  uint32
	Position uint32
}

func readFileHeader(r io.ReadSeeker) (hdr fileHeader, err error) {
	type ewrap struct{ error }
	read := func(data any) {
		err := binary.Read(r, binary.LittleEndian, data)
		if err != nil {
			panic(ewrap{err})
		}
	}
	defer func() {
		switch r := recover().(type) {
		case ewrap:
			err = r.error
		default:
			panic(r)
		}
	}()

	var data uint32
	read(&data)
	if data != magic {
		return hdr, fmt.Errorf("bad magic in header: %x", data)
	}
	read(&hdr.Header)
	read(&hdr.Version)

	var ntocs uint32
	read(ntocs)
	hdr.Tocs = make([]fileToc, ntocs)

	skip := hdr.Header - fileHeaderLen
	if skip > 0 {
		_, err = r.Seek(int64(skip), io.SeekCurrent)
		if err != nil {
			return hdr, fmt.Errorf("seek past header: %w", err)
		}
	}

	for i := range hdr.Tocs {
		read(&hdr.Tocs[i].Type)
		read(&hdr.Tocs[i].Subtype)
		read(&hdr.Tocs[i].Position)
	}

	return hdr, nil
}
