package main

import (
	"flag"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"

	"github.com/mauserzjeh/dxt"
)

func decodeCommand(args []string) {
	decodeFs := flag.NewFlagSet("decode", flag.ExitOnError)
	decodeFs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s decode [options] <-width> <-height> <input DXT file> <output image file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		decodeFs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nIf no format flags are specified, the program will attempt to auto-detect the format based on the filename.\n")
	}
	var (
		width    = decodeFs.Uint("width", 0, "width of the input image (required)")
		height   = decodeFs.Uint("height", 0, "height of the input image (required)")
		readDxt1 = decodeFs.Bool("dxt1", false, "input DXT1 format")
		readDxt3 = decodeFs.Bool("dxt3", false, "input DXT3 format")
		readDxt5 = decodeFs.Bool("dxt5", false, "input DXT5 format")
		writePng = decodeFs.Bool("png", false, "output PNG format")
		writeJpg = decodeFs.Bool("jpg", false, "output JPEG format")
		writeGif = decodeFs.Bool("gif", false, "output GIF format")
	)
	err := decodeFs.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		decodeFs.Usage()
		os.Exit(1)
	}

	if *readDxt1 && *readDxt3 || *readDxt1 && *readDxt5 || *readDxt3 && *readDxt5 {
		fmt.Fprintf(os.Stderr, "error: only one of -dxt1, -dxt3, or -dxt5 can be specified\n")
		decodeFs.Usage()
		os.Exit(1)
	}
	if *writePng && *writeJpg || *writePng && *writeGif || *writeJpg && *writeGif {
		fmt.Fprintf(os.Stderr, "error: only one of -png, -jpg, -gif can be specified\n")
		decodeFs.Usage()
		os.Exit(1)
	}

	input, err := os.ReadFile(decodeFs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input file: %v\n", err)
		os.Exit(2)
	}

	var rgba []byte

	inputFormat := guessFormat(decodeFs.Arg(0))
	if inputFormat == "dxt1" || *readDxt1 {
		rgba, err = dxt.DecodeDXT1(input, *width, *height)
	} else if inputFormat == "dxt3" || *readDxt3 {
		rgba, err = dxt.DecodeDXT3(input, *width, *height)
	} else if inputFormat == "dxt5" || *readDxt5 {
		rgba, err = dxt.DecodeDXT5(input, *width, *height)
	} else {
		fmt.Fprintf(os.Stderr, "error: could not determine input format, please specify -dxt1, -dxt3, or -dxt5\n")
		decodeFs.Usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error decoding input file: %v\n", err)
		os.Exit(3)
	}

	img := image.NewRGBA(image.Rect(0, 0, int(*width), int(*height)))
	img.Pix = rgba

	sink, err := os.Create(decodeFs.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening output file: %v\n", err)
		os.Exit(2)
	}
	defer func() {
		if err := sink.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing output file: %v\n", err)
		}
	}()

	outputFormat := guessFormat(decodeFs.Arg(1))
	if outputFormat == "png" || *writePng {
		err = png.Encode(sink, img)
	} else if outputFormat == "jpg" || *writeJpg {
		err = jpeg.Encode(sink, img, &jpeg.Options{Quality: 100})
	} else if outputFormat == "gif" || *writeGif {
		err = gif.Encode(sink, img, &gif.Options{NumColors: 256})
	} else {
		fmt.Fprintf(os.Stderr, "error: could not determine output format, please specify -png, -jpg, or -gif\n")
		decodeFs.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding output image: %v\n", err)
		os.Exit(3)
	}

	fmt.Printf("Decoded %s as %s to %s as %s (%dx%d)\n", decodeFs.Arg(0), inputFormat, decodeFs.Arg(1), outputFormat, *width, *height)
}
