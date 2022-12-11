package cursor

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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

	var inherits string
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

		if inherits == "" {
			full := filepath.Join(dir, "index.theme")
			inherits = themeInherits(full)
		}
	}
	for path, ok := inherits, inherits != ""; ok; path, ok = nextPath(path) {
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

func themeInherits(full string) (result string, err error) {
	if full == "" {
		return ""
	}

	file, err := os.Open(full)
	if err != nil {
		return "", err
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
		after = strings.TrimSpace(after)
		after = strings.TrimLeftFunc(after, func(c rune) bool {
			switch c {
			case ';', ',':
				return true
			default:
				return unicode.IsSpace(c)
			}
		})

		panic("Not implemented.")
	}
	if err := s.Err(); err != nil {
		return "", fmt.Errorf("scan: %w", err)
	}

	return result, nil
}
