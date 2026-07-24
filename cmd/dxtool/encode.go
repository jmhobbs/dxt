package main

import (
	"flag"
	"fmt"
	"image"
	"os"

	"github.com/mauserzjeh/dxt"
)

func encodeCommand(args []string) {
	encodeFs := flag.NewFlagSet("encode", flag.ExitOnError)
	encodeFs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s encode [options] <input image file> <output DXT file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		encodeFs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nIf no format flags are specified, the program will attempt to auto-detect the format based on the filename.\n")
	}
	var (
		writeDxt1 = encodeFs.Bool("dxt1", false, "output DXT1 format")
		writeDxt3 = encodeFs.Bool("dxt3", false, "output DXT3 format")
		writeDxt5 = encodeFs.Bool("dxt5", false, "output DXT5 format")
	)
	err := encodeFs.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		encodeFs.Usage()
		os.Exit(1)
	}

	if *writeDxt1 && *writeDxt3 || *writeDxt1 && *writeDxt5 || *writeDxt3 && *writeDxt5 {
		fmt.Fprintf(os.Stderr, "error: only one of -dxt1, -dxt3, or -dxt5 can be specified\n")
		encodeFs.Usage()
		os.Exit(1)
	}

	src, err := os.Open(encodeFs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening input file: %v\n", err)
		os.Exit(2)
	}
	defer func() {
		if err := src.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing input file: %v\n", err)
		}
	}()

	img, _, err := image.Decode(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input file: %v\n", err)
		os.Exit(2)
	}

	var buf []byte

	format := guessFormat(encodeFs.Arg(1))
	if format == "dxt1" || *writeDxt1 {
		buf, err = dxt.EncodeDXT1(imgToRGBABytes(img), uint(img.Bounds().Dx()), uint(img.Bounds().Dy()))
	} else if format == "dxt3" || *writeDxt3 {
		buf, err = dxt.EncodeDXT3(imgToRGBABytes(img), uint(img.Bounds().Dx()), uint(img.Bounds().Dy()))
	} else if format == "dxt5" || *writeDxt5 {
		buf, err = dxt.EncodeDXT5(imgToRGBABytes(img), uint(img.Bounds().Dx()), uint(img.Bounds().Dy()))
	} else {
		fmt.Fprintf(os.Stderr, "error: could not determine output format, please specify -dxt1, -dxt3, or -dxt5\n")
		encodeFs.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding image: %v\n", err)
		os.Exit(3)
	}

	err = os.WriteFile(encodeFs.Arg(1), buf, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("Encoded %s to %s as %s (%dx%d)\n", encodeFs.Arg(0), encodeFs.Arg(1), format, img.Bounds().Dx(), img.Bounds().Dy())
}

func imgToRGBABytes(img image.Image) []byte {
	rgb, ok := img.(*image.RGBA)
	if ok {
		return rgb.Pix
	}

	buf := make([]byte, img.Bounds().Dx()*img.Bounds().Dy()*4)
	for y := range img.Bounds().Dy() {
		rowoffset := y * img.Bounds().Dx() * 4
		for x := range img.Bounds().Dx() {
			r, g, b, a := img.At(x, y).RGBA()
			buf[rowoffset+x*4+0] = uint8(r)
			buf[rowoffset+x*4+1] = uint8(g)
			buf[rowoffset+x*4+2] = uint8(b)
			buf[rowoffset+x*4+3] = uint8(a)
		}
	}
	return buf
}
