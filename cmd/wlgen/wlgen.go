package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"

	"deedles.dev/wl/protocol"
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

func main() {
	xmlfile := flag.String("xml", "", "protocol XML file")
	//out := flag.String("out", "", "output file (default <xml file>.go)")
	//pkg := flag.String("pkg", "wl", "output package name")
	//prefix := flag.String("prefix", "wl_", "interface prefix name to strip")
	flag.Parse()

	proto, err := loadXML(*xmlfile)
	if err != nil {
		log.Fatalf("load XML: %v", err)
	}

	fmt.Printf("%+v\n", proto)
}
