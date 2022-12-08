package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"go/build"
	"go/format"
	"log"
	"os"
	"strings"
	"text/template"

	"deedles.dev/wl/internal/set"
	"deedles.dev/wl/protocol"
	"golang.org/x/exp/maps"
)

const baseTmpl = "wlgen.tmpl"

var (
	//go:embed *.tmpl
	tmplFS embed.FS
)

func parseTemplates(ctx Context) *template.Template {
	tmplFuncs := map[string]any{
		"ident":          ctx.ident,
		"camel":          ctx.camel,
		"snake":          ctx.snake,
		"export":         ctx.export,
		"unexport":       ctx.unexport,
		"trimSpace":      strings.TrimSpace,
		"trimLines":      ctx.trimLines,
		"listeners":      ctx.listeners,
		"senders":        ctx.senders,
		"goType":         ctx.goType,
		"typeFuncSuffix": ctx.typeFuncSuffix,
		"unkeyword":      ctx.unkeyword,
		"comment":        ctx.comment,
		"partial":        ctx.partial,
		"args":           ctx.args,
		"returns":        ctx.returns,
		"isRet":          ctx.isRet,
		"package":        ctx.pkg,
		"trimPackage":    ctx.trimPackage,
		"enumType":       ctx.enumType,
	}

	return template.Must(template.New(baseTmpl).Funcs(tmplFuncs).ParseFS(tmplFS, "*.tmpl"))
}

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

type Import struct {
	Prefix string
	Name   string
}

type Config struct {
	Package string
	Prefix  string
	Imports map[string]Import
}

func loadConfig(path string, isClient bool) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	conf := Config{
		Imports: make(map[string]Import),
	}
	var errs []error

	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if (len(line) == 0) || (line[0] == '#') {
			continue
		}

		parts := strings.Fields(line)
		switch parts[0] {
		case "package":
			conf.Package = parts[1]
			if len(parts) == 3 {
				conf.Prefix = parts[2]
			}
		case "import":
			path = parts[1]
			if isClient {
				path = parts[2]
			}
			i := Import{Prefix: parts[3]}
			if len(parts) == 5 {
				i.Name = parts[4]
			}
			if i.Name == "" {
				pkg, err := build.Import(path, "", 0)
				if err != nil {
					errs = append(errs, fmt.Errorf("import %q: %w", path, err))
					conf.Imports[path] = i
					continue
				}
				i.Name = pkg.Name
			}
			conf.Imports[path] = i
		}
	}

	errs = append(errs, s.Err())
	return conf, errors.Join(errs...)
}

type Context struct {
	T            *template.Template
	Protocol     protocol.Protocol
	Config       Config
	IsClient     bool
	Locals       set.Set[string]
	ExtraImports []string
}

func main() {
	xmlfile := flag.String("xml", "", "protocol XML file")
	out := flag.String("out", "", "output file (default <xml file>.go)")
	config := flag.String("config", "", "config file (default <xnl file>.conf)")
	client := flag.Bool("client", false, "generate code for client usage instead of server")
	flag.Parse()

	if *out == "" {
		*out = *xmlfile + ".go"
	}
	if *config == "" {
		*config = *xmlfile + ".conf"
	}

	proto, err := loadXML(*xmlfile)
	if err != nil {
		log.Fatalf("load XML: %v", err)
	}

	conf, err := loadConfig(*config, *client)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := Context{
		Protocol: proto,
		Config:   conf,
		IsClient: *client,
		Locals:   make(set.Set[string]),
	}

	extraImports := make(set.Set[string])
	for _, i := range proto.Interfaces {
		for _, req := range i.Requests {
			for _, arg := range req.Args {
				switch arg.Type {
				case "new_id":
					if arg.Interface != "" {
						ctx.Locals.Add(arg.Interface)
					}
				case "fd":
					extraImports.Add("os")
				}
			}
		}

		for _, ev := range i.Events {
			for _, arg := range ev.Args {
				switch arg.Type {
				case "new_id":
					if arg.Interface != "" {
						ctx.Locals.Add(arg.Interface)
					}
				case "fd":
					extraImports.Add("os")
				}
			}
		}
	}
	ctx.ExtraImports = maps.Keys(extraImports)

	ctx.T = parseTemplates(ctx)

	var buf bytes.Buffer
	err = ctx.T.ExecuteTemplate(&buf, baseTmpl, ctx)
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
