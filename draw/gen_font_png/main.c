#include <stdio.h>
#include <stdlib.h>

#define STB_TRUETYPE_IMPLEMENTATION
#include "stb_truetype.h"

#define STB_IMAGE_WRITE_IMPLEMENTATION
#include "stb_image_write.h"

const float FontHeight = 128.0f;
const int CharsPerRow = 16;
const int RowCount = 16;
const int GlyphPadding = 8;

const int codepoints[] = {
	0, 0x263A, 0x263B, 0x2665, 0x2666, 0x2663, 0x2660, 0x2022, 0x25D8, 0x25CB, 0x25D9, 0x2642, 0x2640, 0x266A, 0x266B, 0x263C,
	0x25BA, 0x25C4, 0x2195, 0x203C, 0x00B6, 0x00A7, 0x25AC, 0x21A8, 0x2191, 0x2193, 0x2192, 0x2190, 0x221F, 0x2194, 0x25B2, 0x25BC,
	' ', '!', '\"', '#', '$', '%', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/',
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', ':', ';', '<', '=', '>', '?',
	'@', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O',
	'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', '[', '\\', ']', 0x5E, '_',
	'`', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o',
	'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '{', '|', '}', '~', 0x2302,
	0xC7, 0xFC, 0xE9, 0xE2, 0xE4, 0xE0, 0xE5, 0xE7, 0xEA, 0xEB, 0xE8, 0xEF, 0xEE, 0xEC, 0xC4, 0xC5,
	0xC8, 0xE6, 0xD6, 0xF6, 0xDC, 0xDF, 0xB2, 0xB3, 0xB4, 0xB0,
};
const int num_codepoints = sizeof(codepoints) / sizeof(codepoints[0]);

unsigned char ttf_buffer[1<<20];
unsigned char temp_bitmap[1024*1024];
stbtt_bakedchar cdata[126 - 32 + 2];

int main() {
	fread(ttf_buffer, 1, 1<<20, fopen("../Go-Mono.ttf", "rb"));

	stbtt_fontinfo font;
	stbtt_InitFont(&font, ttf_buffer, stbtt_GetFontOffsetForIndex(ttf_buffer, 0));

	int ascent, descent, lineGap;
	stbtt_GetFontVMetrics(&font, &ascent, &descent, &lineGap);
	float scale = stbtt_ScaleForPixelHeight(&font, FontHeight);

	int cell_width, cell_height;
	int max_width = 0, max_height = 0;
	int max_ascent = 0, max_descent = 0;

	// Get glyph max dimensions.
	for (int i = 0; i < num_codepoints; i++) {
		int ch = codepoints[i];
		int glyph = 0;
		if(ch != 0)
			glyph = stbtt_FindGlyphIndex(&font, ch);
		int ax, lsb, x0, y0, x1, y1;
		stbtt_GetGlyphHMetrics(&font, glyph, &ax, &lsb);
		stbtt_GetGlyphBitmapBox(&font, glyph, scale, scale, &x0, &y0, &x1, &y1);

		int w = x1 - x0;
		int h = y1 - y0;
		if (w > max_width) max_width = w;
		if (h > max_height) max_height = h;
		if (-y0 > max_ascent) max_ascent = -y0;   // y0 is negative above baseline
		if (y1 > max_descent) max_descent = y1;   // y1 is positive below baseline
	}

	// Add padding.
	cell_width = max_width + GlyphPadding * 2;
	cell_height = max_ascent + max_descent + GlyphPadding * 2;
	int baseline = max_ascent + GlyphPadding;

	int atlas_width = cell_width * CharsPerRow;
	int atlas_height = cell_height * RowCount;
	unsigned char* atlas = calloc(atlas_width * atlas_height, 1);


	for (int i = 0; i < num_codepoints; i++) {
		int ch = codepoints[i];
		int glyph = 0;
		if(ch != 0)
			glyph = stbtt_FindGlyphIndex(&font, ch);
		int xoff, yoff, w, h;
		unsigned char* bitmap = stbtt_GetGlyphBitmap(&font, 0, scale, glyph, &w, &h, &xoff, &yoff);

		// Calculate cell top-left corner.
		int cx = (i % CharsPerRow) * cell_width;
		int cy = (i / CharsPerRow) * cell_height;

		// Position the glyph at the baseline within the cell.
		int x_pos = cx + (cell_width - w) / 2;  // horizontal centering
		int y_pos = cy + baseline + yoff;	   // vertical alignment to baseline

		for (int row = 0; row < h; row++) {
			memcpy(atlas + (y_pos + row) * atlas_width + x_pos, bitmap + row * w, w);
		}
	}

	stbi_write_png("font.png", atlas_width, atlas_height, 1, atlas, atlas_width);

	return 0;
}
