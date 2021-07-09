package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gonutz/prototype/draw"
)

const (
	gameW                   = 10
	gameH                   = 18
	startHeight             = gameH - 2
	deadHeight              = gameH - 1
	tileSize                = 30
	scoreOffset             = 20
	dropSpeedAccelaration   = 3
	initialDropSpeed        = 60
	minDropSpeed            = 5
	linesBeforeAcceleration = 10
)

func main() {
	rand.Seed(time.Now().UnixNano())
	scores := []int{0, 10, 25, 100, 500}

	var (
		cur               *block
		next              *block
		dropDelay         int
		nextSpeedIncrease int
		nextDrop          int
		field             *field
		score             int
		totalLines        int
		paused            bool
	)

	dropBlock := func() (hitGround bool) {
		cur.y--
		if cur.stuckIn(field) {
			cur.y++
			field.solidify(cur)
			cur = next
			next = randomBlock()
			nextDrop = dropDelay
			return true
		}
		return false
	}

	dropBlockAllTheWay := func() {
		for !dropBlock() {
		}
	}

	restart := func() {
		cur = randomBlock()
		next = randomBlock()
		dropDelay = initialDropSpeed
		nextSpeedIncrease = linesBeforeAcceleration
		nextDrop = dropDelay
		field = newField(gameW, gameH)
		score = 0
		totalLines = 0
		paused = false
	}

	restart()

	mainErr := draw.RunWindow("Blocks", (5+gameW)*tileSize, gameH*tileSize+scoreOffset,
		func(window draw.Window) {
			if window.WasKeyPressed(draw.KeyEscape) {
				window.Close()
				return
			}

			if window.WasKeyPressed(draw.KeyF2) {
				restart()
				return
			}

			lost := field.isGameOver()

			if !lost {
				if window.WasKeyPressed(draw.KeyP) {
					paused = !paused
				}
				if !paused {
					if window.WasKeyPressed(draw.KeyDown) {
						dropBlock()
					}
					if window.WasKeyPressed(draw.KeySpace) {
						dropBlockAllTheWay()
					}
					if window.WasKeyPressed(draw.KeyUp) {
						if window.IsKeyDown(draw.KeyLeftShift) ||
							window.IsKeyDown(draw.KeyRightShift) {
							cur.rotate(field, -1)
						} else {
							cur.rotate(field, 1)
						}
					}
					if window.WasKeyPressed(draw.KeyLeft) {
						cur.x--
						if cur.stuckIn(field) {
							cur.x++
						}
					}
					if window.WasKeyPressed(draw.KeyRight) {
						cur.x++
						if cur.stuckIn(field) {
							cur.x--
						}
					}

					nextDrop--
					if nextDrop == 0 {
						nextDrop = dropDelay
						dropBlock()
					}

					removed := field.clearLines()
					totalLines += removed
					score += scores[removed]
					if totalLines >= nextSpeedIncrease {
						nextSpeedIncrease += linesBeforeAcceleration
						dropDelay -= dropSpeedAccelaration
						if dropDelay < minDropSpeed {
							dropDelay = minDropSpeed
						}
					}
				}
			}
			// compute where the block would end up if dropped to the ground
			dropped := *cur
			for !dropped.stuckIn(field) {
				dropped.y--
			}
			dropped.y++

			window.FillRect(
				gameW*tileSize,
				0,
				5*tileSize,
				gameH*tileSize+scoreOffset,
				draw.DarkGray,
			)

			if !paused {
				field.draw(window)
				dropped.draw(window, 0.15)
				cur.draw(window, 1)
				next.drawAt(window, gameW*tileSize-tileSize-tileSize/2, 3*tileSize, 1)
			}
			window.FillRect(0, 0, gameW*tileSize, scoreOffset, draw.White)
			scoreText := fmt.Sprintf("Score %v  Lines %v", score, totalLines)
			w, h := window.GetTextSize(scoreText)
			x, y := (gameW*tileSize-w)/2, (scoreOffset-h)/2
			window.DrawText(scoreText, x, y, draw.Black)

			text := ""
			if lost {
				text = "Game Over"
			} else if paused {
				text = "Pause"
			}
			if text != "" {
				scale := float32(3.0)
				w, h := window.GetScaledTextSize(text, scale)
				x, y := (gameW*tileSize-w)/2, (gameH*tileSize-h)/2
				window.FillRect(x, y, w, h, draw.Black)
				window.DrawScaledText(text, x, y, scale, draw.White)
			}

			window.DrawText(`
Up: rotate
Space: drop
P:  (un)pause
F2: new game`, gameW*tileSize+15, gameH*tileSize+scoreOffset-85, draw.White)
		})

	if mainErr != nil {
		panic(mainErr)
	}
}

type block struct {
	rotation int
	shape    shape
	x, y     int
}

// direction +1 means rotate right, -1 means left
func (b *block) rotate(field *field, direction int) {
	oldRotation := b.rotation
	b.rotation = (b.rotation + len(tileOffsets[b.shape]) + direction) % len(tileOffsets[b.shape])
	if b.stuckIn(field) {
		b.rotation = oldRotation
	}
}

func (b *block) drawAt(window draw.Window, ofsX, ofsY int, alpha float32) {
	color := shapeColors[b.shape]
	color.A = alpha
	outlineColor := color
	outlineColor.R *= 0.5
	outlineColor.G *= 0.5
	outlineColor.B *= 0.5
	tiles := b.tiles()
	for _, tile := range tiles {
		x, y := ofsX+tile.x*tileSize, ofsY+tileToScreenY(tile.y)
		window.FillRect(x, y, tileSize, tileSize, color)
		window.DrawRect(x, y, tileSize, tileSize, outlineColor)
	}
}

func (b *block) draw(window draw.Window, alpha float32) {
	b.drawAt(window, 0, 0, alpha)
}

func (b *block) tiles() (tiles [4]point) {
	offsets := tileOffsets[b.shape][b.rotation]
	for i := range tiles {
		tiles[i].x = b.x + offsets[i].x
		tiles[i].y = b.y + offsets[i].y
	}
	return
}

var tileOffsets = [shapeCount][][4]point{
	// L
	{
		{{0, 0}, {1, 0}, {0, 1}, {0, 2}},
		{{-1, 0}, {-1, 1}, {0, 1}, {1, 1}},
		{{0, 0}, {0, 2}, {0, 1}, {-1, 2}},
		{{1, 1}, {1, 2}, {0, 1}, {-1, 1}},
	},
	// J
	{
		{{0, 0}, {1, 0}, {1, 1}, {1, 2}},
		{{0, 1}, {0, 2}, {1, 1}, {2, 1}},
		{{1, 0}, {1, 2}, {1, 1}, {2, 2}},
		{{0, 1}, {2, 1}, {1, 1}, {2, 0}},
	},
	// O
	{
		{{0, 0}, {0, 1}, {1, 0}, {1, 1}},
	},
	// I
	{
		{{0, 0}, {0, 1}, {0, 2}, {0, 3}},
		{{-1, 1}, {0, 1}, {1, 1}, {2, 1}},
	},
	// S
	{
		{{0, 0}, {1, 0}, {1, 1}, {2, 1}},
		{{1, 0}, {1, 1}, {0, 1}, {0, 2}},
	},
	// Z
	{
		{{0, 1}, {1, 1}, {1, 0}, {2, 0}},
		{{0, 0}, {0, 1}, {1, 1}, {1, 2}},
	},
	// T
	{
		{{1, 0}, {0, 1}, {1, 1}, {2, 1}},
		{{0, 1}, {1, 1}, {1, 2}, {1, 0}},
		{{1, 2}, {0, 1}, {1, 1}, {2, 1}},
		{{2, 1}, {1, 1}, {1, 2}, {1, 0}},
	},
}

func (b *block) stuckIn(field *field) bool {
	for _, tile := range b.tiles() {
		if tile.x < 0 || tile.x >= field.w {
			return true
		}
		if tile.y < 0 {
			return true
		}
		if field.getTile(tile.x, tile.y) != -1 {
			return true
		}
	}
	return false
}

func newField(w, h int) *field {
	emptyTiles := make([]int, w*h)
	for i := range emptyTiles {
		emptyTiles[i] = -1
	}
	return &field{w: w, h: h, tiles: emptyTiles}
}

type field struct {
	w, h  int
	tiles []int
}

func (f *field) solidify(b *block) {
	for _, tile := range b.tiles() {
		f.setTile(tile.x, tile.y, int(b.shape))
	}
}

func (f *field) getTile(x, y int) int {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return empty
	}
	index := x + y*f.w
	return f.tiles[index]
}

const empty = -1

func (f *field) setTile(x, y int, to int) {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return
	}
	index := x + y*f.w
	f.tiles[index] = to
}

func (f *field) draw(window draw.Window) {
	for y := 0; y < f.h; y++ {
		for x := 0; x < f.w; x++ {
			tile := f.getTile(x, y)
			color := draw.Black
			if tile != empty {
				color = shapeColors[tile]
			}
			window.FillRect(x*tileSize, tileToScreenY(y), tileSize, tileSize, color)
		}
	}
}

func (f *field) clearLines() (removed int) {
	for y := f.h - 1; y >= 0; y-- {
		if f.lineFull(y) {
			f.removeLine(y)
			removed++
		}
	}
	return
}

func (f *field) lineFull(y int) bool {
	for x := 0; x < f.w; x++ {
		if f.getTile(x, y) == empty {
			return false
		}
	}
	return true
}

func (f *field) removeLine(y int) {
	for drop := y + 1; drop < f.h; drop++ {
		for x := 0; x < f.w; x++ {
			f.setTile(x, drop-1, f.getTile(x, drop))
		}
	}
}

func (f *field) isGameOver() bool {
	for y := deadHeight; y < gameH; y++ {
		for x := 0; x < gameW; x++ {
			if f.getTile(x, y) != empty {
				return true
			}
		}
	}
	return false
}

type point struct{ x, y int }

type shape int

// We name the shapes by their form:
//
//     x         x       xx       x        xx       xx        xxx
// L = x    J =  x   O = xx   I = x   S = xx    Z =  xx   T =  x
//     xx       xx                x
//                                x
const (
	L shape = iota
	J
	O
	I
	S
	Z
	T
	shapeCount
)

var shapes = [shapeCount]shape{L, J, O, I, S, Z, T}

var shapeColors = [shapeCount]draw.Color{
	draw.Red,
	draw.LightBlue,
	draw.Green,
	draw.Yellow,
	draw.LightBrown,
	draw.Cyan,
	draw.White,
}

func randomShape() shape {
	return shapes[rand.Intn(len(shapes))]
}

func randomBlock() *block {
	return &block{
		shape: randomShape(),
		x:     gameW/2 - 2,
		y:     startHeight,
	}
}

func tileToScreenY(tileY int) int {
	return scoreOffset + (gameH-tileY-1)*tileSize
}
