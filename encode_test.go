package dxt

import (
	"io/ioutil"
	"testing"
)

// abs returns the absolute value of a signed int difference
func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func assertNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// makeRGBA builds a width*height*4 RGBA buffer using the given generator
func makeRGBA(width, height uint, gen func(x, y uint) (r, g, b, a byte)) []byte {
	out := make([]byte, width*height*4)
	for y := uint(0); y < height; y++ {
		for x := uint(0); x < width; x++ {
			idx := (y*width + x) * 4
			r, g, b, a := gen(x, y)
			out[idx+0] = r
			out[idx+1] = g
			out[idx+2] = b
			out[idx+3] = a
		}
	}
	return out
}

// assertChannelsClose fails if any RGBA channel in got/want differs by more than tolerance
func assertChannelsClose(t testing.TB, got, want []byte, width, height uint, tolerance uint32) {
	t.Helper()
	assertEqual(t, len(got), len(want))
	for y := uint(0); y < height; y++ {
		for x := uint(0); x < width; x++ {
			idx := (y*width + x) * 4
			for c := uint(0); c < 4; c++ {
				g := uint32(got[idx+c])
				w := uint32(want[idx+c])
				if absDiff(g, w) > tolerance {
					t.Fatalf("pixel (%d,%d) channel %d: got %d, want %d (tolerance %d)", x, y, c, g, w, tolerance)
				}
			}
		}
	}
}

// --- helper unit tests ---

func TestPack565RoundTrip(t *testing.T) {
	for c := 0; c < 65536; c++ {
		r, g, b := unpack_565(uint32(c))
		packed := pack_565(r, g, b)
		assertEqual(t, packed, uint16(c))
	}
}

// --- DXT1 ---

func TestEncodeDXT1SolidColor(t *testing.T) {
	width, height := uint(8), uint(8)
	img := makeRGBA(width, height, func(x, y uint) (byte, byte, byte, byte) {
		return 200, 100, 50, 255
	})

	enc, err := EncodeDXT1(img, width, height)
	assertNoError(t, err)

	dec, err := DecodeDXT1(enc, width, height)
	assertNoError(t, err)

	assertChannelsClose(t, dec, img, width, height, 10)
}

func TestEncodeDXT1IdempotentAgainstFixture(t *testing.T) {
	enc, err := ioutil.ReadFile("testdata/dxt1.encoded")
	assertNoError(t, err)

	imgA, err := DecodeDXT1(enc, 256, 256)
	assertNoError(t, err)

	reencoded, err := EncodeDXT1(imgA, 256, 256)
	assertNoError(t, err)

	imgB, err := DecodeDXT1(reencoded, 256, 256)
	assertNoError(t, err)

	assertChannelsClose(t, imgB, imgA, 256, 256, 24)
}

func TestEncodeDXT1PartialBlockGradient(t *testing.T) {
	width, height := uint(6), uint(7)
	img := makeRGBA(width, height, func(x, y uint) (byte, byte, byte, byte) {
		v := x*20 + y*15
		return byte(v), byte(v / 2), 128, 255
	})

	enc, err := EncodeDXT1(img, width, height)
	assertNoError(t, err)
	assertEqual(t, len(enc), int(((width+3)/4)*((height+3)/4)*8))

	dec, err := DecodeDXT1(enc, width, height)
	assertNoError(t, err)

	assertChannelsClose(t, dec, img, width, height, 20)
}

func TestEncodeDXT1Transparency(t *testing.T) {
	width, height := uint(4), uint(4)
	img := makeRGBA(width, height, func(x, y uint) (byte, byte, byte, byte) {
		if x < 2 {
			return 10, 20, 30, 0
		}
		return 220, 210, 200, 255
	})

	enc, err := EncodeDXT1(img, width, height)
	assertNoError(t, err)

	dec, err := DecodeDXT1(enc, width, height)
	assertNoError(t, err)

	for y := uint(0); y < height; y++ {
		for x := uint(0); x < width; x++ {
			idx := (y*width + x) * 4
			if x < 2 {
				assertEqual(t, dec[idx+0], byte(0))
				assertEqual(t, dec[idx+1], byte(0))
				assertEqual(t, dec[idx+2], byte(0))
				assertEqual(t, dec[idx+3], byte(0))
			} else {
				assertEqual(t, dec[idx+3], byte(255))
			}
		}
	}
}

func TestEncodeDXT1InvalidInputLength(t *testing.T) {
	_, err := EncodeDXT1(make([]byte, 10), 4, 4)
	assertEqual(t, err != nil, true)
}

// --- DXT3 ---

func TestEncodeDXT3SolidColor(t *testing.T) {
	width, height := uint(8), uint(8)
	img := makeRGBA(width, height, func(x, y uint) (byte, byte, byte, byte) {
		return 10, 220, 130, 170
	})

	enc, err := EncodeDXT3(img, width, height)
	assertNoError(t, err)

	dec, err := DecodeDXT3(enc, width, height)
	assertNoError(t, err)

	assertChannelsClose(t, dec, img, width, height, 10)
}

func TestEncodeDXT3AlphaGridExact(t *testing.T) {
	// alpha values that sit exactly on the 4-bit quantization grid (multiples of 17)
	// must survive round trip exactly.
	width, height := uint(4), uint(4)
	img := makeRGBA(width, height, func(x, y uint) (byte, byte, byte, byte) {
		return 128, 64, 32, byte((x + y*4) * 17)
	})

	enc, err := EncodeDXT3(img, width, height)
	assertNoError(t, err)

	dec, err := DecodeDXT3(enc, width, height)
	assertNoError(t, err)

	for y := uint(0); y < height; y++ {
		for x := uint(0); x < width; x++ {
			idx := (y*width + x) * 4
			assertEqual(t, dec[idx+3], img[idx+3])
		}
	}
}

func TestEncodeDXT3IdempotentAgainstFixture(t *testing.T) {
	enc, err := ioutil.ReadFile("testdata/dxt3.encoded")
	assertNoError(t, err)

	imgA, err := DecodeDXT3(enc, 128, 512)
	assertNoError(t, err)

	reencoded, err := EncodeDXT3(imgA, 128, 512)
	assertNoError(t, err)

	imgB, err := DecodeDXT3(reencoded, 128, 512)
	assertNoError(t, err)

	assertChannelsClose(t, imgB, imgA, 128, 512, 24)
}

func TestEncodeDXT3PartialBlockGradient(t *testing.T) {
	width, height := uint(6), uint(7)
	img := makeRGBA(width, height, func(x, y uint) (byte, byte, byte, byte) {
		v := x*20 + y*15
		return byte(v), byte(v / 2), 128, byte(x*20 + y*10)
	})

	enc, err := EncodeDXT3(img, width, height)
	assertNoError(t, err)

	dec, err := DecodeDXT3(enc, width, height)
	assertNoError(t, err)

	assertChannelsClose(t, dec, img, width, height, 20)
}

func TestEncodeDXT3InvalidInputLength(t *testing.T) {
	_, err := EncodeDXT3(make([]byte, 10), 4, 4)
	assertEqual(t, err != nil, true)
}

// --- DXT5 ---

func TestEncodeDXT5SolidColor(t *testing.T) {
	width, height := uint(8), uint(8)
	img := makeRGBA(width, height, func(x, y uint) (byte, byte, byte, byte) {
		return 30, 60, 90, 120
	})

	enc, err := EncodeDXT5(img, width, height)
	assertNoError(t, err)

	dec, err := DecodeDXT5(enc, width, height)
	assertNoError(t, err)

	assertChannelsClose(t, dec, img, width, height, 10)
}

func TestEncodeDXT5AlphaExtremesExact(t *testing.T) {
	// a block containing an exact 0 and 255 alpha forces 6-level mode,
	// which must preserve those two values exactly.
	width, height := uint(4), uint(4)
	img := makeRGBA(width, height, func(x, y uint) (byte, byte, byte, byte) {
		a := byte(128)
		if x == 0 && y == 0 {
			a = 0
		}
		if x == 3 && y == 3 {
			a = 255
		}
		return 100, 100, 100, a
	})

	enc, err := EncodeDXT5(img, width, height)
	assertNoError(t, err)

	dec, err := DecodeDXT5(enc, width, height)
	assertNoError(t, err)

	assertEqual(t, dec[(0*width+0)*4+3], byte(0))
	assertEqual(t, dec[(3*width+3)*4+3], byte(255))
}

func TestEncodeDXT5IdempotentAgainstFixture(t *testing.T) {
	enc, err := ioutil.ReadFile("testdata/dxt5.encoded")
	assertNoError(t, err)

	imgA, err := DecodeDXT5(enc, 64, 64)
	assertNoError(t, err)

	reencoded, err := EncodeDXT5(imgA, 64, 64)
	assertNoError(t, err)

	imgB, err := DecodeDXT5(reencoded, 64, 64)
	assertNoError(t, err)

	assertChannelsClose(t, imgB, imgA, 64, 64, 24)
}

func TestEncodeDXT5PartialBlockGradient(t *testing.T) {
	width, height := uint(6), uint(7)
	img := makeRGBA(width, height, func(x, y uint) (byte, byte, byte, byte) {
		v := x*20 + y*15
		return byte(v), byte(v / 2), 128, byte(30 + x*20 + y*10)
	})

	enc, err := EncodeDXT5(img, width, height)
	assertNoError(t, err)

	dec, err := DecodeDXT5(enc, width, height)
	assertNoError(t, err)

	assertChannelsClose(t, dec, img, width, height, 20)
}

func TestEncodeDXT5InvalidInputLength(t *testing.T) {
	_, err := EncodeDXT5(make([]byte, 10), 4, 4)
	assertEqual(t, err != nil, true)
}
