package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"

	_ "image/gif"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// PosterLongEdge is the largest width or height a stored poster keeps.
const PosterLongEdge = 600

// ResizePoster decodes an image (JPEG, PNG, GIF or WebP), turns a camera
// photo upright (its EXIF orientation), shrinks it so its long edge is at
// most PosterLongEdge, and re-encodes it as JPEG. Transparent areas become
// white.
func ResizePoster(raw []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, errors.New("empty image")
	}
	if long := max(w, h); long > PosterLongEdge {
		w, h = max(1, w*PosterLongEdge/long), max(1, h*PosterLongEdge/long)
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	// Turned after shrinking (the long edge stays the same), so a large
	// camera photo is turned at poster size.
	img := upright(dst, jpegOrientation(raw))
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// jpegOrientation reads a JPEG's EXIF orientation (1 to 8; 1 is upright).
// Anything else, or anything it can't read, counts as upright.
func jpegOrientation(b []byte) int {
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(b); {
		if b[i] != 0xFF {
			return 1
		}
		marker := b[i+1]
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			i += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 { // image data: no EXIF before it
			return 1
		}
		n := int(binary.BigEndian.Uint16(b[i+2:]))
		end := i + 2 + n
		if n < 2 || end > len(b) {
			return 1
		}
		if marker == 0xE1 && n >= 8 && string(b[i+4:i+10]) == "Exif\x00\x00" {
			return exifOrientation(b[i+10 : end])
		}
		i = end
	}
	return 1
}

// exifOrientation finds tag 0x0112 in the first IFD of a TIFF block.
func exifOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var o binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		o = binary.LittleEndian
	case "MM":
		o = binary.BigEndian
	default:
		return 1
	}
	ifd := int(o.Uint32(t[4:]))
	if ifd < 8 || ifd+2 > len(t) {
		return 1
	}
	count := int(o.Uint16(t[ifd:]))
	for k := 0; k < count; k++ {
		e := ifd + 2 + 12*k
		if e+12 > len(t) {
			return 1
		}
		if o.Uint16(t[e:]) == 0x0112 {
			if v := int(o.Uint16(t[e+8:])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// upright applies an EXIF orientation: the returned image looks the way
// the camera meant it to.
func upright(src image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if orientation >= 5 { // a quarter turn swaps width and height
		w, h = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			var dx, dy int
			switch orientation {
			case 2: // mirrored
				dx, dy = b.Dx()-1-x, y
			case 3: // upside down
				dx, dy = b.Dx()-1-x, b.Dy()-1-y
			case 4: // mirrored upside down
				dx, dy = x, b.Dy()-1-y
			case 5: // mirrored, quarter turn left
				dx, dy = y, x
			case 6: // needs a quarter turn clockwise
				dx, dy = b.Dy()-1-y, x
			case 7: // mirrored, quarter turn right
				dx, dy = b.Dy()-1-y, b.Dx()-1-x
			case 8: // needs a quarter turn anticlockwise
				dx, dy = y, b.Dx()-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
