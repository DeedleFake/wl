package main

import (
	"fmt"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"deedles.dev/wl/internal/xslices"
	"deedles.dev/wl/protocol"
)

func (ctx Context) ident(v string) string {
	var pkg string
	v, ok := strings.CutPrefix(v, ctx.Config.Prefix)
	if !ok {
		for _, i := range ctx.Config.Imports {
			// TODO: Figure out how to make this work with multiple
			// protocols with the same prefix.
			v, ok = strings.CutPrefix(v, i.Prefix)
			if ok {
				pkg = i.Name + "."
				break
			}
		}
	}

	return pkg + ctx.camel(v)
}

func (ctx Context) camel(v string) string {
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
}

func (ctx Context) snake(v string) string {
	var buf strings.Builder
	buf.Grow(len(v))
	for i, c := range v {
		if unicode.IsUpper(c) && (i > 0) {
			buf.WriteRune('_')
		}
		buf.WriteRune(unicode.ToLower(c))
	}
	return buf.String()
}

func (ctx Context) export(v string) string {
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
}

func (ctx Context) unexport(v string) string {
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
}

func (ctx Context) trimLines(v string) string {
	lines := strings.Split(v, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}

func (ctx Context) listeners(i protocol.Interface) []protocol.Op {
	if ctx.IsClient {
		return i.Events
	}
	return i.Requests
}

func (ctx Context) senders(i protocol.Interface) []protocol.Op {
	if ctx.IsClient {
		return i.Requests
	}
	return i.Events
}

func (ctx Context) goType(arg protocol.Arg) (string, error) {
	switch arg.Type {
	case "uint":
		return "uint32", nil
	case "int":
		return "int32", nil
	case "fixed":
		return "wire.Fixed", nil
	case "object":
		if arg.Interface == "" {
			return "uint32", nil
		}
		return "*" + ctx.ident(arg.Interface), nil
	case "new_id":
		if arg.Interface == "" {
			return "wire.NewID", nil
		}
		return "uint32", nil
	case "string":
		return "string", nil
	case "array":
		return "[]byte", nil
	case "fd":
		return "*os.File", nil
	default:
		return "", fmt.Errorf("unknown type: %q", arg.Type)
	}
}

func (ctx Context) typeFuncSuffix(arg protocol.Arg) (string, error) {
	switch arg.Type {
	case "uint":
		return "Uint", nil
	case "int":
		return "Int", nil
	case "fixed":
		return "Fixed", nil
	case "object":
		if arg.Interface != "" {
			return "Object", nil
		}
		return "Uint", nil
	case "new_id":
		if arg.Interface == "" {
			return "NewID", nil
		}
		return "Uint", nil
	case "string":
		return "String", nil
	case "array":
		return "Array", nil
	case "fd":
		return "File", nil
	default:
		return "", fmt.Errorf("unknown type: %q", arg.Type)
	}
}

func (ctx Context) unkeyword(v string) string {
	if token.IsKeyword(v) {
		return "_" + v
	}
	return v
}

func (ctx Context) comment(v string) string {
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
}

func (ctx Context) partial(name string, data any) (string, error) {
	var sb strings.Builder
	err := ctx.T.ExecuteTemplate(&sb, name, data)
	return sb.String(), err
}

func (ctx Context) args(op protocol.Op) (args []protocol.Arg) {
	return xslices.Filter(op.Args, func(arg protocol.Arg) bool { return !ctx.isRet(arg) })
}

func (ctx Context) returns(op protocol.Op) (rets []protocol.Arg) {
	return xslices.Filter(op.Args, ctx.isRet)
}

func (ctx Context) isRet(arg protocol.Arg) bool {
	return (arg.Type == "new_id") && (arg.Interface != "")
}

func (ctx Context) pkg(v string) string {
	before, _, ok := strings.Cut(v, ".")
	if ok {
		return before + "."
	}
	return ""
}

func (ctx Context) trimPackage(v string) string {
	before, after, ok := strings.Cut(v, ".")
	if ok {
		return after
	}
	return before
}
