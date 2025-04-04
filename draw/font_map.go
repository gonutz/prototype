package draw

// Each letter in the font bitmap has a border of fontGlyphMargin around it, to
// each side. So the total margin left + right is fontGlyphMargin * 2.
const fontGlyphMargin = 1

// runeToFont maps a unicode rune to the index of the respective glyph in the
// font bitmap. The bitmap contains only a subset of all existing runes, if r is
// not present in the bitmap, a replacement character is returned.
func runeToFont(r rune) rune {
	if 0 <= r && r <= 127 {
		return r
	}
	return fontMap[r]
}

var fontMap = map[rune]rune{
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
	'§': 150,
	'²': 151,
	'³': 152,
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
