package draw

import (
	"image"
	"unsafe"

	_ "embed"
)

//go:embed font.png
var bitmapFontWhitePng []byte

// Each letter in the font bitmap has a border of fontGlyphMargin around it, to
// each side. So the total margin left + right is fontGlyphMargin * 2.
const fontGlyphMargin = 8

// fontBaseScale is the scale factor to use for the regular text size used by
// draw.Window.DrawText.
const fontBaseScale = 1.0 / 8

// fontKerningFactor is an extra scale factor for the width of each character.
// This makes the desktop font the same size as the WASM font.
const fontKerningFactor = 0.97

// runeToFont maps a unicode rune to the index of the respective glyph in the
// font bitmap. The bitmap contains only a subset of all existing runes, if r is
// not present in the bitmap, a replacement character is returned.
func runeToFont(r rune) rune {
	if 32 <= r && r <= 127 {
		return r
	}
	return fontMap[r]
}

var fontMap = map[rune]rune{
	'☺': 1,
	'☻': 2,
	'♥': 3,
	'♦': 4,
	'♣': 5,
	'♠': 6,
	'•': 7,
	'◘': 8,
	'○': 9,
	'◙': 10,
	'♂': 11,
	'♀': 12,
	'♪': 13,
	'♫': 14,
	'☼': 15,
	'►': 16,
	'◄': 17,
	'↕': 18,
	'‼': 19,
	'¶': 20,
	'§': 21,
	'▬': 22,
	'↨': 23,
	'↑': 24,
	'↓': 25,
	'→': 26,
	'←': 27,
	'∟': 28,
	'↔': 29,
	'▲': 30,
	'▼': 31,
	'Ç': 128,
	'ü': 129,
	'é': 130,
	'â': 131,
	'ä': 132,
	'à': 133,
	'å': 134,
	'ç': 135,
	'ê': 136,
	'ë': 137,
	'ё': 137,
	'è': 138,
	'ѐ': 138,
	'Ї': 139,
	'Ï': 139,
	'Î': 140,
	'Ì': 141,
	'Ä': 142,
	'Å': 143,
	'È': 144,
	'Ѐ': 144,
	'æ': 145,
	'Ö': 146,
	'ö': 147,
	'Ü': 148,
	'ß': 149,
	'²': 150,
	'³': 151,
	'´': 152,
	'°': 153,
	// Cyrillic letters that look like existing ones.
	'Ѕ': 'S',
	'І': 'I',
	'Ј': 'J',
	'А': 'A',
	'В': 'B',
	'Е': 'E',
	'З': '3',
	'К': 'K',
	'М': 'M',
	'Н': 'H',
	'О': 'O',
	'Р': 'P',
	'С': 'C',
	'Т': 'T',
	'У': 'y',
	'Х': 'X',
	'Ь': 'b',
	'а': 'a',
	'в': 'B',
	'г': 'r',
	'е': 'e',
	'з': '3',
	'к': 'K',
	'м': 'M',
	'н': 'H',
	'о': 'o',
	'р': 'p',
	'с': 'c',
	'т': 'T',
	'у': 'y',
	'х': 'x',
	'ъ': 'b',
	'ь': 'b',
	'ѕ': 's',
	'і': 'i',
	'ј': 'j',
	'ѡ': 'w',
	'Ѵ': 'V',
	'ѵ': 'v',
}

// nextFontTextureMipMap returns an image half the size of img in both
// directions. It is intended to create the font texture mipmaps up to four
// levels from the original. Its pixels will all be white with an alpha value
// that is the combination of the four corresponding pixels in img. The mipmaps
// are made brighter instead of just averaging the four input pixels. This is
// necessary to avoid the font getting darker as it gets smaller because a fully
// white (255) pixel in img might have a fully transparent (0) neighbor, which
// get combined into (255+0)/2 = 127 which is much darker. So we brighten the
// mipmaps by a factor figured out from trial and error, which makes the font
// look nice for all sizes.
func nextFontTextureMipMap(img *image.NRGBA) *image.NRGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w/2, h/2))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := img.PixOffset(x, y)
			// We increase alpha for smaller images so that the font does not
			// get darker and darker as it gets smaller.
			a := uint32(img.Pix[i+3]) * 14 / 10
			destI := out.PixOffset(x/2, y/2)
			*(*uint32)(unsafe.Pointer(&out.Pix[destI])) += a
		}
	}

	for y := 0; y < h/2; y++ {
		for x := 0; x < w/2; x++ {
			i := out.PixOffset(x, y)
			sum := *(*uint32)(unsafe.Pointer(&out.Pix[i]))
			a := uint8(min(255, (sum+2)/4))
			out.Pix[i+0] = 255
			out.Pix[i+1] = 255
			out.Pix[i+2] = 255
			out.Pix[i+3] = a
		}
	}

	return out
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
