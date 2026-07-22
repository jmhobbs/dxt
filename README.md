![GitHub release (latest by date)](https://img.shields.io/github/v/release/mauserzjeh/dxt?style=flat-square)

# dxt

DXT compression library written in Go. It supports DXT1, DXT3 and DXT5 compression from, and
decompression to, RGBA

# Installation
```
go get -u github.com/mauserzjeh/dxt
```

# Tests
```
go test -v
```

# Usage
```go
// import the library
import "github.com/mauserzjeh/dxt"

var dxtBytes []byte
var width uint
var height uint

// ...read the DXT encoded data...
// ...and also obtain the width and height of the image...

// decompress DXT1 to RGBA
rgbaBytes, err := dxt.DecodeDXT1(dxtBytes, width, height)

// or

// decompress DXT3 to RGBA
rgbaBytes, err := dxt.DecodeDXT3(dxtBytes, width, height)

// or

// decompress DXT5 to RGBA
rgbaBytes, err := dxt.DecodeDXT5(dxtBytes, width, height)

// check for errors
if err != nil {
    log.Fatal(err)
}

// rgbaBytes should hold the decompressed RGBA data if no error happened
//             R    G   B    A    R   G    B  ...
// ie. []byte{123, 23, 234, 212, 21, 128, 52, ...}
```

```go
// import the library
import "github.com/mauserzjeh/dxt"

var rgbaBytes []byte
var width uint
var height uint

// ...fill rgbaBytes with RGBA data (4 bytes per pixel)...
// ...and set width/height to the image's dimensions...

// compress RGBA to DXT1
dxtBytes, err := dxt.EncodeDXT1(rgbaBytes, width, height)

// or

// compress RGBA to DXT3
dxtBytes, err := dxt.EncodeDXT3(rgbaBytes, width, height)

// or

// compress RGBA to DXT5
dxtBytes, err := dxt.EncodeDXT5(rgbaBytes, width, height)

// check for errors
if err != nil {
    log.Fatal(err)
}

// dxtBytes should hold the compressed data if no error happened
```
