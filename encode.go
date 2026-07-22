package dxt

import (
	"fmt"
)

// pixel holds a single RGBA texel while a 4x4 block is being encoded
type pixel struct {
	r, g, b, a uint32
}

// pack_565 packs 8 bit RGB components into a 565 color, the inverse of unpack_565
func pack_565(r, g, b uint32) uint16 {
	r5 := (r*31 + 127) / 255
	g6 := (g*63 + 127) / 255
	b5 := (b*31 + 127) / 255

	return uint16(r5<<11 | g6<<5 | b5)
}

// colorPalette builds the 4 color DXT palette for a pair of 565 endpoints,
// using the exact same formulas the decoders use so encoded indices decode
// back to precisely these colors
func colorPalette(c0raw, c1raw uint16) [4]uint32 {
	c0 := uint32(c0raw)
	c1 := uint32(c1raw)

	r0, g0, b0 := unpack_565(c0)
	r1, g1, b1 := unpack_565(c1)

	var palette [4]uint32
	palette[0] = pack_rgba(r0, g0, b0, 255)
	palette[1] = pack_rgba(r1, g1, b1, 255)
	palette[2] = pack_rgba(c2(r0, r1, c0, c1), c2(g0, g1, c0, c1), c2(b0, b1, c0, c1), 255)
	if c0 > c1 {
		palette[3] = pack_rgba(c3(r0, r1), c3(g0, g1), c3(b0, b1), 255)
	} else {
		palette[3] = pack_rgba(0, 0, 0, 0)
	}

	return palette
}

// nearestColorIndex finds the palette entry closest to r,g,b by squared distance
func nearestColorIndex(r, g, b uint32, palette [4]uint32) uint32 {
	best := uint32(0)
	bestDist := int64(-1)

	for i := 0; i < 4; i++ {
		pr, pg, pb, _ := unpack_rgba(palette[i])
		dr := int64(r) - int64(pr)
		dg := int64(g) - int64(pg)
		db := int64(b) - int64(pb)
		dist := dr*dr + dg*dg + db*db

		if bestDist == -1 || dist < bestDist {
			bestDist = dist
			best = uint32(i)
		}
	}

	return best
}

// rangeFitEndpoints picks two real pixel colors from the block as endpoint
// candidates: the pixels at the extremes of whichever channel has the
// largest spread. This is a simple, deterministic baseline color fit.
func rangeFitEndpoints(pixels []pixel) (uint16, uint16) {
	minR, minG, minB := pixels[0].r, pixels[0].g, pixels[0].b
	maxR, maxG, maxB := pixels[0].r, pixels[0].g, pixels[0].b

	for _, p := range pixels[1:] {
		if p.r < minR {
			minR = p.r
		}
		if p.g < minG {
			minG = p.g
		}
		if p.b < minB {
			minB = p.b
		}
		if p.r > maxR {
			maxR = p.r
		}
		if p.g > maxG {
			maxG = p.g
		}
		if p.b > maxB {
			maxB = p.b
		}
	}

	rSpread := maxR - minR
	gSpread := maxG - minG
	bSpread := maxB - minB

	channel := 0
	if gSpread > rSpread {
		channel = 1
	}
	if bSpread > rSpread && bSpread > gSpread {
		channel = 2
	}

	channelValue := func(p pixel) uint32 {
		switch channel {
		case 0:
			return p.r
		case 1:
			return p.g
		default:
			return p.b
		}
	}

	lo, hi := pixels[0], pixels[0]
	loV, hiV := channelValue(pixels[0]), channelValue(pixels[0])

	for _, p := range pixels[1:] {
		v := channelValue(p)
		if v < loV {
			loV = v
			lo = p
		}
		if v > hiV {
			hiV = v
			hi = p
		}
	}

	return pack_565(hi.r, hi.g, hi.b), pack_565(lo.r, lo.g, lo.b)
}

// fitColorEndpoints fits endpoints for pixels and orders them as raw 565
// words to force either 4-color opaque mode (c0 > c1) or 3-color mode
// (c0 <= c1), nudging apart equal endpoints when opaque mode is forced
// since c0 > c1 must hold strictly.
func fitColorEndpoints(pixels []pixel, forceOpaque bool) (uint16, uint16) {
	c0, c1 := rangeFitEndpoints(pixels)

	if forceOpaque {
		if c0 <= c1 {
			c0, c1 = c1, c0
		}
		if c0 == c1 {
			if c1 > 0 {
				c1--
			} else {
				c0++
			}
		}
	} else if c0 > c1 {
		c0, c1 = c1, c0
	}

	return c0, c1
}

// packColorIndices assigns each of a block's 16 pixels to a palette entry
// and packs the 2 bit indices in the same order the decoders read them.
// forceIndex3, if non-nil, forces a given pixel to index 3 (used for
// DXT1's punch-through transparent pixels) instead of nearest matching.
func packColorIndices(block [16]pixel, palette [4]uint32, forceIndex3 func(i int) bool) uint32 {
	var bitcode uint32

	for i := 15; i >= 0; i-- {
		var idx uint32
		if forceIndex3 != nil && forceIndex3(i) {
			idx = 3
		} else {
			idx = nearestColorIndex(block[i].r, block[i].g, block[i].b, palette)
		}
		bitcode = (bitcode << 2) | idx
	}

	return bitcode
}

// extractBlock reads a 4x4 block of pixels starting at block coordinates
// (bx, by), replicating the nearest real pixel to pad blocks that run past
// the image edge.
func extractBlock(input []byte, width, height, bx, by uint) [16]pixel {
	var block [16]pixel

	for row := uint(0); row < 4; row++ {
		srcY := by*4 + row
		if srcY >= height {
			srcY = height - 1
		}
		for col := uint(0); col < 4; col++ {
			srcX := bx*4 + col
			if srcX >= width {
				srcX = width - 1
			}
			idx := (srcY*width + srcX) * 4
			block[row*4+col] = pixel{
				r: uint32(input[idx+0]),
				g: uint32(input[idx+1]),
				b: uint32(input[idx+2]),
				a: uint32(input[idx+3]),
			}
		}
	}

	return block
}

// nonTransparentPixels returns the block's pixels not flagged transparent,
// falling back to a single black pixel if every pixel is transparent
func nonTransparentPixels(block [16]pixel, transparent [16]bool) []pixel {
	pixels := make([]pixel, 0, 16)
	for i, p := range block {
		if !transparent[i] {
			pixels = append(pixels, p)
		}
	}
	if len(pixels) == 0 {
		pixels = append(pixels, pixel{})
	}
	return pixels
}

// dxtAlphaThreshold is the alpha value below which a pixel is treated as
// transparent for DXT1's punch-through alpha mode
const dxtAlphaThreshold = 128

// validateEncodeInput checks that input holds exactly one RGBA byte per pixel
func validateEncodeInput(input []byte, width, height uint) error {
	want := width * height * 4
	if uint(len(input)) != want {
		return fmt.Errorf("dxt: invalid input length: got %d, want %d", len(input), want)
	}
	return nil
}

// EncodeDXT1 encodes an RGBA byte slice to DXT1. Blocks with any pixel
// whose alpha is below 128 are encoded in punch-through alpha mode, where
// those pixels decode back to fully transparent black; all other blocks
// are encoded fully opaque.
func EncodeDXT1(input []byte, width, height uint) (output []byte, err error) {
	if err := validateEncodeInput(input, width, height); err != nil {
		return nil, err
	}

	block_count_x := (width + 3) / 4
	block_count_y := (height + 3) / 4
	output = make([]byte, block_count_x*block_count_y*8)

	offset := uint(0)
	for by := uint(0); by < block_count_y; by++ {
		for bx := uint(0); bx < block_count_x; bx++ {
			block := extractBlock(input, width, height, bx, by)

			var transparent [16]bool
			hasTransparency := false
			for i, p := range block {
				if p.a < dxtAlphaThreshold {
					transparent[i] = true
					hasTransparency = true
				}
			}

			var c0raw, c1raw uint16
			if hasTransparency {
				c0raw, c1raw = fitColorEndpoints(nonTransparentPixels(block, transparent), false)
			} else {
				c0raw, c1raw = fitColorEndpoints(block[:], true)
			}

			palette := colorPalette(c0raw, c1raw)

			var indices uint32
			if hasTransparency {
				indices = packColorIndices(block, palette, func(i int) bool { return transparent[i] })
			} else {
				indices = packColorIndices(block, palette, nil)
			}

			output[offset+0] = byte(c0raw)
			output[offset+1] = byte(c0raw >> 8)
			output[offset+2] = byte(c1raw)
			output[offset+3] = byte(c1raw >> 8)
			output[offset+4] = byte(indices)
			output[offset+5] = byte(indices >> 8)
			output[offset+6] = byte(indices >> 16)
			output[offset+7] = byte(indices >> 24)

			offset += 8
		}
	}

	return output, nil
}

// encodeAlphaBlockDXT3 quantizes a block's alpha channel to 4 bits per
// pixel, packed 4 pixels per row matching DecodeDXT3's unpack order
func encodeAlphaBlockDXT3(block [16]pixel) [4]uint16 {
	var words [4]uint16

	for row := uint(0); row < 4; row++ {
		var word uint16
		for col := uint(0); col < 4; col++ {
			a4 := (block[row*4+col].a*15 + 127) / 255
			word |= uint16(a4) << (col * 4)
		}
		words[row] = word
	}

	return words
}

// EncodeDXT3 encodes an RGBA byte slice to DXT3
func EncodeDXT3(input []byte, width, height uint) (output []byte, err error) {
	if err := validateEncodeInput(input, width, height); err != nil {
		return nil, err
	}

	block_count_x := (width + 3) / 4
	block_count_y := (height + 3) / 4
	output = make([]byte, block_count_x*block_count_y*16)

	offset := uint(0)
	for by := uint(0); by < block_count_y; by++ {
		for bx := uint(0); bx < block_count_x; bx++ {
			block := extractBlock(input, width, height, bx, by)

			alphaWords := encodeAlphaBlockDXT3(block)

			c0raw, c1raw := fitColorEndpoints(block[:], true)
			palette := colorPalette(c0raw, c1raw)
			indices := packColorIndices(block, palette, nil)

			for i, word := range alphaWords {
				output[offset+uint(i)*2+0] = byte(word)
				output[offset+uint(i)*2+1] = byte(word >> 8)
			}

			output[offset+8] = byte(c0raw)
			output[offset+9] = byte(c0raw >> 8)
			output[offset+10] = byte(c1raw)
			output[offset+11] = byte(c1raw >> 8)
			output[offset+12] = byte(indices)
			output[offset+13] = byte(indices >> 8)
			output[offset+14] = byte(indices >> 16)
			output[offset+15] = byte(indices >> 24)

			offset += 16
		}
	}

	return output, nil
}

// nearestAlphaLevel finds the index of the level closest to a
func nearestAlphaLevel(a uint32, levels [8]uint32) uint32 {
	best := uint32(0)
	bestDist := int64(-1)

	for i := 0; i < 8; i++ {
		d := int64(a) - int64(levels[i])
		if d < 0 {
			d = -d
		}
		if bestDist == -1 || d < bestDist {
			bestDist = d
			best = uint32(i)
		}
	}

	return best
}

// encodeAlphaBlockDXT5 picks DXT5's 8-level interpolation mode (a0 > a1)
// or, when the block contains an exact 0 or 255 alpha that needs to be
// preserved exactly, the 6-level + explicit 0/255 mode (a0 <= a1),
// matching DecodeDXT5's two branches, and packs the 48 bit index field
// at the alignment DecodeDXT5 reads (bytes offset+2..offset+7).
func encodeAlphaBlockDXT5(block [16]pixel) (a0, a1 byte, indices uint64) {
	var alphas [16]uint32
	hasExtreme := false
	for i, p := range block {
		alphas[i] = p.a
		if p.a == 0 || p.a == 255 {
			hasExtreme = true
		}
	}

	var levels [8]uint32
	var e0, e1 uint32

	if hasExtreme {
		lo, hi := uint32(255), uint32(0)
		found := false
		for _, a := range alphas {
			if a == 0 || a == 255 {
				continue
			}
			found = true
			if a < lo {
				lo = a
			}
			if a > hi {
				hi = a
			}
		}
		if !found {
			lo, hi = 0, 255
		}

		e0, e1 = lo, hi
		levels[0] = e0
		levels[1] = e1
		levels[2] = (e0*4 + e1*1) / 5
		levels[3] = (e0*3 + e1*2) / 5
		levels[4] = (e0*2 + e1*3) / 5
		levels[5] = (e0*1 + e1*4) / 5
		levels[6] = 0
		levels[7] = 255
	} else {
		minA, maxA := alphas[0], alphas[0]
		for _, a := range alphas[1:] {
			if a < minA {
				minA = a
			}
			if a > maxA {
				maxA = a
			}
		}

		e0, e1 = maxA, minA
		if e0 == e1 {
			if e1 > 0 {
				e1--
			} else {
				e0++
			}
		}

		levels[0] = e0
		levels[1] = e1
		levels[2] = (e0*6 + e1*1) / 7
		levels[3] = (e0*5 + e1*2) / 7
		levels[4] = (e0*4 + e1*3) / 7
		levels[5] = (e0*3 + e1*4) / 7
		levels[6] = (e0*2 + e1*5) / 7
		levels[7] = (e0*1 + e1*6) / 7
	}

	var bits uint64
	for i := 15; i >= 0; i-- {
		idx := nearestAlphaLevel(alphas[i], levels)
		bits = (bits << 3) | uint64(idx)
	}

	return byte(e0), byte(e1), bits
}

// EncodeDXT5 encodes an RGBA byte slice to DXT5
func EncodeDXT5(input []byte, width, height uint) (output []byte, err error) {
	if err := validateEncodeInput(input, width, height); err != nil {
		return nil, err
	}

	block_count_x := (width + 3) / 4
	block_count_y := (height + 3) / 4
	output = make([]byte, block_count_x*block_count_y*16)

	offset := uint(0)
	for by := uint(0); by < block_count_y; by++ {
		for bx := uint(0); bx < block_count_x; bx++ {
			block := extractBlock(input, width, height, bx, by)

			a0, a1, alphaIndices := encodeAlphaBlockDXT5(block)

			c0raw, c1raw := fitColorEndpoints(block[:], true)
			palette := colorPalette(c0raw, c1raw)
			colorIndices := packColorIndices(block, palette, nil)

			output[offset+0] = a0
			output[offset+1] = a1
			output[offset+2] = byte(alphaIndices)
			output[offset+3] = byte(alphaIndices >> 8)
			output[offset+4] = byte(alphaIndices >> 16)
			output[offset+5] = byte(alphaIndices >> 24)
			output[offset+6] = byte(alphaIndices >> 32)
			output[offset+7] = byte(alphaIndices >> 40)

			output[offset+8] = byte(c0raw)
			output[offset+9] = byte(c0raw >> 8)
			output[offset+10] = byte(c1raw)
			output[offset+11] = byte(c1raw >> 8)
			output[offset+12] = byte(colorIndices)
			output[offset+13] = byte(colorIndices >> 8)
			output[offset+14] = byte(colorIndices >> 16)
			output[offset+15] = byte(colorIndices >> 24)

			offset += 16
		}
	}

	return output, nil
}
