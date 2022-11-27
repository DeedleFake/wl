// Package protocol defines the types necessary for unmarshalling a
// protocol-specification XML file.
package protocol

type Protocol struct {
	Name      string `xml:"name,attr"`
	Copyright string `xml:"copyright"`

	Interfaces []Interface `xml:"interface"`
}

type Interface struct {
	Name        string      `xml:"name,attr"`
	Version     int         `xml:"version,attr"`
	Description Description `xml:"description"`

	Requests []Op   `xml:"request"`
	Events   []Op   `xml:"event"`
	Enums    []Enum `xml:"enum"`
}

type Description struct {
	Summary string `xml:"summary,attr"`
	Full    string `xml:",chardata"`
}

type Op struct {
	Name        string      `xml:"name,attr"`
	Description Description `xml:"description"`

	Args []Arg `xml:"arg"`
}

type Arg struct {
	Name    string `xml:"name,attr"`
	Summary string `xml:"summary,attr"`

	Type      string `xml:"type,attr"`
	Interface string `xml:"interface,attr"`
	Version   int    `xml:"version,attr"`
}

type Enum struct {
	Name        string      `xml:"name,attr"`
	Description Description `xml:"description"`

	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Name    string `xml:"name,attr"`
	Summary string `xml:"summary,attr"`
	Value   int    `xml:"value,attr"`
}
