package main

import (
	"bytes"
	"embed"
	"encoding/xml"
	"flag"
	"fmt"
	"go/format"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"deedles.dev/wl/protocol"
)

var (
	//go:embed *.tmpl
	tmplFS embed.FS
	tmpl   = template.Must(template.New("base").Funcs(tmplFuncs).ParseFS(tmplFS, "*.tmpl"))

	tmplFuncs = map[string]any{
		"camel": func(v string) string {
			var buf strings.Builder
			buf.Grow(len(v))
			shift := true
			for _, c := range v {
				if c == '_' {
					shift = true
					continue
				}

				if shift {
					c = unicode.ToUpper(c)
				}
				buf.WriteRune(c)
				shift = false
			}
			return buf.String()
		},
		"snake": func(v string) string {
			var buf strings.Builder
			buf.Grow(len(v))
			for i, c := range v {
				if unicode.IsUpper(c) && (i > 0) {
					buf.WriteRune('_')
				}
				buf.WriteRune(unicode.ToLower(c))
			}
			return buf.String()
		},
		"export": func(v string) string {
			if len(v) == 0 {
				return ""
			}

			c, size := utf8.DecodeRuneInString(v)
			if unicode.IsUpper(c) {
				return v
			}

			var buf strings.Builder
			buf.Grow(len(v))
			buf.WriteRune(unicode.ToUpper(c))
			buf.WriteString(v[size:])
			return buf.String()
		},
		"unexport": func(v string) string {
			if len(v) == 0 {
				return ""
			}

			c, size := utf8.DecodeRuneInString(v)
			if unicode.IsLower(c) {
				return v
			}

			var buf strings.Builder
			buf.Grow(len(v))
			buf.WriteRune(unicode.ToLower(c))
			buf.WriteString(v[size:])
			return buf.String()
		},
		"trimPrefix": func(prefix, v string) string {
			return strings.TrimPrefix(v, prefix)
		},
		"trimSpace": strings.TrimSpace,
		"trimLines": func(v string) string {
			lines := strings.Split(v, "\n")
			for i := range lines {
				lines[i] = strings.TrimSpace(lines[i])
			}
			return strings.Join(lines, "\n")
		},
		"listeners": func(isClient bool, i protocol.Interface) []protocol.Op {
			if isClient {
				return i.Events
			}
			return i.Requests
		},
		"senders": func(isClient bool, i protocol.Interface) []protocol.Op {
			if isClient {
				return i.Requests
			}
			return i.Events
		},
		"typeFuncSuffix": func(arg protocol.Arg) (string, error) {
			switch arg.Type {
			case "uint", "object":
				return "Uint", nil
			case "new_id":
				if arg.Interface == "" {
					return "NewID", nil
				}
				return "Uint", nil
			case "int":
				return "Int", nil
			case "fixed":
				return "Fixed", nil
			case "fd":
				return "File", nil
			case "string":
				return "String", nil
			case "array":
				return "Array", nil
			default:
				return "", fmt.Errorf("unknown type: %q", arg.Type)
			}
		},
		"goType": func(arg protocol.Arg) (string, error) {
			switch arg.Type {
			case "uint", "object":
				return "uint32", nil
			case "new_id":
				if arg.Interface == "" {
					return "wire.NewID", nil
				}
				return "uint32", nil
			case "int":
				return "int32", nil
			case "fixed":
				return "wire.Fixed", nil
			case "fd":
				return "*os.File", nil
			case "string":
				return "string", nil
			case "array":
				return "[]byte", nil
			default:
				return "", fmt.Errorf("unknown type: %q", arg.Type)
			}
		},
		"unkeyword": func(v string) string {
			if token.IsKeyword(v) {
				return "_" + v
			}
			return v
		},
		"comment": func(v string) string {
			if len(v) == 0 {
				return ""
			}

			var sb strings.Builder
			for _, line := range strings.Split(v, "\n") {
				sb.WriteString("// ")
				sb.WriteString(line)
				sb.WriteByte('\n')
			}
			return sb.String()
		},
	}
)

func loadXML(path string) (proto protocol.Protocol, err error) {
	file, err := os.Open(path)
	if err != nil {
		return proto, err
	}
	defer file.Close()

	d := xml.NewDecoder(file)
	err = d.Decode(&proto)
	return proto, err
}

type TemplateContext struct {
	Protocol     protocol.Protocol
	Package      string
	Prefix       string
	IsClient     bool
	ExtraImports []string
}

func main() {
	xmlfile := flag.String("xml", "", "protocol XML file")
	out := flag.String("out", "", "output file (default <xml file>.go)")
	pkg := flag.String("pkg", "wl", "output package name")
	prefix := flag.String("prefix", "wl_", "interface prefix name to strip")
	client := flag.Bool("client", false, "generate code for client usage instead of server")
	flag.Parse()

	if *out == "" {
		*out = *xmlfile + ".go"
	}

	proto, err := loadXML(*xmlfile)
	if err != nil {
		log.Fatalf("load XML: %v", err)
	}

	var extraImports []string
extraImportsLoop:
	for _, i := range proto.Interfaces {
		for _, req := range i.Requests {
			for _, arg := range req.Args {
				if arg.Type == "fd" {
					extraImports = append(extraImports, "os")
					break extraImportsLoop
				}
			}
		}
		for _, ev := range i.Events {
			for _, arg := range ev.Args {
				if arg.Type == "fd" {
					extraImports = append(extraImports, "os")
					break extraImportsLoop
				}
			}
		}
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "main.tmpl", TemplateContext{
		Protocol:     proto,
		Package:      *pkg,
		Prefix:       *prefix,
		IsClient:     *client,
		ExtraImports: extraImports,
	})
	if err != nil {
		log.Fatalf("execute template: %v", err)
	}

	unfmt := buf.Bytes()
	data, err := format.Source(unfmt)
	if err != nil {
		log.Printf("format output: %v", err)
		data = unfmt
	}

	file, err := os.Create(*out)
	if err != nil {
		log.Fatalf("create output file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		log.Fatalf("write output: %v", err)
	}
}
