package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [encode|decode]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "	encode   Encode to DXT1/3/5\n")
		fmt.Fprintf(os.Stderr, "	decode   Decode from DXT1/3/5\n")
		fmt.Fprintf(os.Stderr, "\nThis tool does not support reading/writing DDS files, only raw DXT1/3/5 data.\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	switch flag.Arg(0) {
	case "encode":
		encodeCommand(flag.Args()[1:])
	case "decode":
		decodeCommand(flag.Args()[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: invalid command\n")
		fmt.Fprintln(os.Stderr, "")
		flag.Usage()
		os.Exit(1)
	}
}

func guessFormat(filename string) string {
	fmt := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	if fmt == "jpeg" {
		return "jpg"
	}
	return fmt
}
