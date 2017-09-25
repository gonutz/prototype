package main

import (
	"math/rand"
	"strings"
	"unicode/utf8"

	"github.com/gonutz/prototype/draw"
)

var (
	wallColor       = draw.LightBlue
	cookieColor     = draw.LightYellow
	playerColor     = draw.Yellow
	backgroundColor = draw.Black
)

const (
	tileSize    = 21
	playerSpeed = tileSize / 8.4
)

func main() {
	level := parseLevel(level1)

	py := float64(level.playerStartY * tileSize)
	px := float64(level.playerStartX * tileSize)
	if level.playerStartDir == left {
		px -= tileSize / 2
	} else if level.playerStartDir == right {
		px += tileSize / 2
	}
	player := player{
		character: character{
			x:       px,
			y:       py,
			facing:  level.playerStartDir,
			nextDir: level.playerStartDir,
		},
	}

	var ghosts []ghost
	for x := 0; x < level.width; x++ {
		for y := 0; y < level.height; y++ {
			tile := level.at(x, y)
			switch tile {
			case ghostStartRed, ghostStartBrown, ghostStartGreen, ghostStartPink:
				color := draw.Red
				if tile == ghostStartBrown {
					color = draw.LightBrown
				} else if tile == ghostStartGreen {
					color = draw.DarkGreen
				} else if tile == ghostStartPink {
					color = draw.Purple
				}
				ghosts = append(ghosts, ghost{
					character: character{
						x:      float64(x) * tileSize,
						y:      float64(y) * tileSize,
						facing: randomDir(),
					},
					color: color,
				})
			}
		}
	}

	const mouthOpenDelay = 10
	mouthOpenTimer := mouthOpenDelay

	windowW, windowH := level.width*tileSize, level.height*tileSize
	draw.RunWindow(
		"Cookies",
		windowW,
		windowH,
		func(window draw.Window) {
			// handle input
			if window.WasKeyPressed(draw.KeyEscape) {
				window.Close()
			}
			if window.WasKeyPressed(draw.KeyLeft) {
				player.move(left)
			}
			if window.WasKeyPressed(draw.KeyRight) {
				player.move(right)
			}
			if window.WasKeyPressed(draw.KeyUp) {
				player.move(up)
			}
			if window.WasKeyPressed(draw.KeyDown) {
				player.move(down)
			}
			if window.IsKeyDown(draw.KeyLeft) {
				player.move(left)
			}
			if window.IsKeyDown(draw.KeyRight) {
				player.move(right)
			}
			if window.IsKeyDown(draw.KeyUp) {
				player.move(up)
			}
			if window.IsKeyDown(draw.KeyDown) {
				player.move(down)
			}

			// update world
			moveCharacter(&player.character, &level)
			tx, ty := player.centerTile()
			if level.at(tx, ty) == cookie || level.at(tx, ty) == bigCookie {
				// TODO play cool sound here
				level.set(tx, ty, space)
			}
			if level.cookieCount == 0 {
				window.Close()
			}

			mouthOpenTimer--
			if mouthOpenTimer < 0 {
				mouthOpenTimer = mouthOpenDelay
				player.mouthOpen = !player.mouthOpen
			}

			// draw everything
			drawWorld(&level, &player, ghosts, window)
		})
}

type character struct {
	x, y    float64
	moving  bool
	facing  faceDir
	nextDir faceDir
}

type player struct {
	character
	mouthOpen bool
}

func (p *player) move(dir faceDir) {
	p.moving = true
	p.nextDir = dir
}

func (p *player) centerTile() (tx, ty int) {
	return round(p.x+tileSize/2) / tileSize, round(p.y+tileSize/2) / tileSize
}

type ghost struct {
	character
	color draw.Color
}

type faceDir int

const (
	left faceDir = iota
	up
	right
	down
)

func randomDir() faceDir {
	return faceDir(rand.Intn(4))
}

func opposite(a, b faceDir) bool {
	switch a {
	case left:
		return b == right
	case right:
		return b == left
	case up:
		return b == down
	case down:
		return b == up
	}
	return false
}

func parseLevel(s string) level {
	var lev level
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	lev.height = len(lines)
	lev.width = utf8.RuneCountInString(lines[0])
	lev.tiles = make([]tile, lev.width*lev.height)
	for y, line := range lines {
		x := -1
		for _, tile := range line {
			x++
			switch tile {
			case ' ':
				lev.set(x, y, space)
			case '─':
				lev.set(x, y, wallEW)
			case '│':
				lev.set(x, y, wallNS)
			case '┌':
				lev.set(x, y, wallES)
			case '┐':
				lev.set(x, y, wallSW)
			case '└':
				lev.set(x, y, wallNE)
			case '┘':
				lev.set(x, y, wallWN)
			case '·':
				lev.set(x, y, cookie)
			case 'O':
				lev.set(x, y, bigCookie)
			case '/', '\\':
				lev.set(x, y, door)
			case ')', '(':
				lev.set(x, y, space)
				lev.playerStartX = x
				lev.playerStartY = y
				lev.playerStartDir = left
				if tile == '(' {
					lev.playerStartDir = right
				}
			case '?':
				lev.set(x, y, specialSpot)
			case 'R':
				lev.set(x, y, ghostStartRed)
			case 'G':
				lev.set(x, y, ghostStartGreen)
			case 'B':
				lev.set(x, y, ghostStartBrown)
			case 'P':
				lev.set(x, y, ghostStartPink)
			default:
				panic("unknown level character: " + string(tile))
			}
		}
	}
	return lev
}

type level struct {
	width, height  int
	tiles          []tile
	playerStartX   int
	playerStartY   int
	playerStartDir faceDir
	cookieCount    int
}

type tile int

const (
	space tile = iota
	outside
	wallEW
	wallNS
	wallES
	wallSW
	wallNE
	wallWN
	cookie
	bigCookie
	door
	specialSpot
	ghostStartRed
	ghostStartBrown
	ghostStartGreen
	ghostStartPink
)

func (t tile) solid() bool {
	return wallEW <= t && t <= wallWN
}

func (t tile) solidOrOutside() bool {
	return t == outside || wallEW <= t && t <= wallWN
}

func (t tile) isCookie() bool {
	return t == cookie || t == bigCookie
}

func (l *level) tileIndex(x, y int) int {
	return x + y*l.width
}

func (l *level) set(x, y int, tile tile) {
	i := l.tileIndex(x, y)
	if l.tiles[i].isCookie() && !tile.isCookie() {
		l.cookieCount--
	}
	if !l.tiles[i].isCookie() && tile.isCookie() {
		l.cookieCount++
	}
	l.tiles[i] = tile
}

func (l *level) at(x, y int) tile {
	if x < 0 || y < 0 || x >= l.width || y >= l.height {
		return outside
	}
	return l.tiles[l.tileIndex(x, y)]
}

func drawWorld(l *level, p *player, ghosts []ghost, window draw.Window) {
	w, h := window.Size()
	window.FillRect(0, 0, w, h, backgroundColor)
	drawPlayer(p, l, window)
	for _, g := range ghosts {
		drawGhost(round(g.x), round(g.y), g.color, g.facing, window)
	}
	drawLevel(l, window)
}

func drawPlayer(p *player, level *level, window draw.Window) {
	drawAt := func(px, py, d int) {
		window.FillEllipse(px-d, py-d, tileSize+2*d, tileSize+2*d, playerColor)
		if p.mouthOpen {
			cx, cy := px+tileSize/2, py+tileSize/2
			switch p.facing {
			case left:
				x := px - d
				for y := py - d; y < py-d+tileSize+2*d; y++ {
					window.DrawLine(x, y, cx, cy, backgroundColor)
				}
			case right:
				x := px - d + tileSize + 2*d
				for y := py - d; y < py-d+tileSize+2*d; y++ {
					window.DrawLine(x, y, cx, cy, backgroundColor)
				}
			case up:
				y := py - d
				for x := px - d; x < px-d+tileSize+2*d; x++ {
					window.DrawLine(x, y, cx, cy, backgroundColor)
				}
			case down:
				y := py - d + tileSize + 2*d
				for x := px - d; x < px-d+tileSize+2*d; x++ {
					window.DrawLine(x, y, cx, cy, backgroundColor)
				}
			}
		}
	}

	d := tileSize / 4
	px, py := round(p.x), round(p.y)
	drawAt(px, py, d)
	if px-d < 0 {
		drawAt(px+level.width*tileSize, py, d)
	}
	if px+tileSize+d >= level.width*tileSize {
		drawAt(px-level.width*tileSize, py, d)
	}
	if py-d < 0 {
		drawAt(px, py+level.height*tileSize, d)
	}
	if py+tileSize+d >= level.height*tileSize {
		drawAt(px, py-level.height*tileSize, d)
	}
}

// TODO make ghost drawing work for all resolutions
func drawGhost(x, y int, color draw.Color, dir faceDir, window draw.Window) {
	window.FillRect(x-1, y+13, 27, 14, color) // connect top and bottom
	window.FillEllipse(x, y, 25, 25, color)   // head
	// four bottom circles
	window.FillEllipse(x-1, y+25, 5, 5, color)
	window.FillEllipse(x+6, y+25, 5, 5, color)
	window.FillEllipse(x+14, y+25, 5, 5, color)
	window.FillEllipse(x+21, y+25, 5, 5, color)
	// eyes
	switch dir {
	case down:
		window.FillEllipse(x+6, y+14, 4, 6, draw.White)
		window.FillEllipse(x+15, y+14, 4, 6, draw.White)
		window.FillEllipse(x+6, y+16, 4, 4, draw.Black)
		window.FillEllipse(x+15, y+16, 4, 4, draw.Black)
	case up:
		window.FillEllipse(x+6, y+8, 4, 6, draw.White)
		window.FillEllipse(x+15, y+8, 4, 6, draw.White)
		window.FillEllipse(x+6, y+8, 4, 4, draw.Black)
		window.FillEllipse(x+15, y+8, 4, 4, draw.Black)
	case left:
		window.FillEllipse(x+4, y+12, 4, 6, draw.White)
		window.FillEllipse(x+13, y+12, 4, 6, draw.White)
		window.FillEllipse(x+3, y+13, 4, 4, draw.Black)
		window.FillEllipse(x+12, y+13, 4, 4, draw.Black)
	case right:
		window.FillEllipse(x+8, y+12, 4, 6, draw.White)
		window.FillEllipse(x+17, y+12, 4, 6, draw.White)
		window.FillEllipse(x+9, y+13, 4, 4, draw.Black)
		window.FillEllipse(x+18, y+13, 4, 4, draw.Black)
	}
}

func drawLevel(l *level, window draw.Window) {
	for ty := 0; ty < l.height; ty++ {
		for tx := 0; tx < l.width; tx++ {
			tile := l.at(tx, ty)
			x, y := tx*tileSize, ty*tileSize
			switch tile {
			case wallES:
				window.DrawLine(
					x+tileSize-1,
					y+tileSize/2,
					x+tileSize/2,
					y+tileSize-1,
					wallColor,
				)
			case wallEW:
				window.DrawLine(
					x,
					y+tileSize/2,
					x+tileSize-1,
					y+tileSize/2,
					wallColor,
				)
			case wallNE:
				window.DrawLine(
					x+tileSize/2,
					y,
					x+tileSize-1,
					y+(tileSize-1)/2,
					wallColor,
				)
			case wallNS:
				window.DrawLine(
					x+tileSize/2,
					y,
					x+tileSize/2,
					y+tileSize-1,
					wallColor,
				)
			case wallSW:
				window.DrawLine(
					x,
					y+tileSize/2,
					x+(tileSize-1)/2,
					y+tileSize-1,
					wallColor,
				)
			case wallWN:
				window.DrawLine(
					x,
					y+tileSize/2,
					x+tileSize/2,
					y,
					wallColor,
				)
			case cookie:
				d := tileSize / 3
				window.FillEllipse(x+d, y+d, tileSize-2*d, tileSize-2*d, cookieColor)
			case bigCookie:
				window.FillEllipse(x, y, tileSize, tileSize, cookieColor)
			case door:
				window.FillRect(x, y, tileSize, tileSize, draw.LightGreen)
			}
		}
	}
}

func moveCharacter(p *character, lev *level) {
	if playerSpeed > tileSize {
		panic("this collision detection will not work for such fast player speeds")
	}

	if !p.moving || opposite(p.facing, p.nextDir) {
		p.facing = p.nextDir
	}
	switch p.facing {
	case left:
		tx := round(p.x) / tileSize
		newTx := round(p.x-playerSpeed) / tileSize
		if tx == newTx {
			p.x -= playerSpeed
		} else {
			ty := round(p.y) / tileSize
			dx := -playerSpeed
			restDx := float64(tx)*tileSize - (p.x + dx)
			if p.nextDir != left {
				if p.nextDir == up && !lev.at(tx, ty-1).solidOrOutside() {
					p.facing = up
					p.x = float64(tx) * tileSize
					p.y = float64(ty)*tileSize - restDx
				} else if p.nextDir == down && !lev.at(tx, ty+1).solidOrOutside() {
					p.facing = down
					p.x = float64(tx) * tileSize
					p.y = float64(ty)*tileSize + restDx
				} else {
					p.x = float64(tx) * tileSize
					if !lev.at(newTx, ty).solid() {
						p.x -= restDx
					}
				}
			} else {
				p.x = float64(tx) * tileSize
				if !lev.at(newTx, ty).solid() {
					p.x -= restDx
				}
			}
		}
	case up:
		ty := round(p.y) / tileSize
		newTy := round(p.y-playerSpeed) / tileSize
		if ty == newTy {
			p.y -= playerSpeed
		} else {
			tx := round(p.x) / tileSize
			dy := -playerSpeed
			restDy := float64(ty)*tileSize - (p.y + dy)
			if p.nextDir != up {
				if p.nextDir == left && !lev.at(tx-1, ty).solidOrOutside() {
					p.facing = left
					p.y = float64(ty) * tileSize
					p.x = float64(tx)*tileSize - restDy
				} else if p.nextDir == right && !lev.at(tx+1, ty).solidOrOutside() {
					p.facing = right
					p.y = float64(ty) * tileSize
					p.x = float64(tx)*tileSize + restDy
				} else {
					p.y = float64(ty) * tileSize
					if !lev.at(tx, newTy).solid() {
						p.y -= restDy
					}
				}
			} else {
				p.y = float64(ty) * tileSize
				if !lev.at(tx, newTy).solid() {
					p.y -= restDy
				}
			}
		}
	case right:
		tx := round(p.x+tileSize-1) / tileSize
		newTx := round(p.x+tileSize-1+playerSpeed) / tileSize
		if tx == newTx {
			p.x += playerSpeed
		} else {
			ty := round(p.y) / tileSize
			dx := playerSpeed
			restDx := (p.x + tileSize - 1 + dx) - float64(newTx)*tileSize
			if p.nextDir != right {
				if p.nextDir == up && !lev.at(tx, ty-1).solidOrOutside() {
					p.facing = up
					p.x = float64(tx) * tileSize
					p.y = float64(ty)*tileSize - restDx
				} else if p.nextDir == down && !lev.at(tx, ty+1).solidOrOutside() {
					p.facing = down
					p.x = float64(tx) * tileSize
					p.y = float64(ty)*tileSize + restDx
				} else {
					p.x = float64(tx) * tileSize
					if !lev.at(newTx, ty).solid() {
						p.x += restDx
					}
				}
			} else {
				p.x = float64(tx) * tileSize
				if !lev.at(newTx, ty).solid() {
					p.x += restDx
				}
			}
		}
	case down:
		ty := round(p.y+tileSize-1) / tileSize
		newTy := round(p.y+tileSize-1+playerSpeed) / tileSize
		if ty == newTy {
			p.y += playerSpeed
		} else {
			tx := round(p.x) / tileSize
			dy := playerSpeed
			restDy := (p.y + tileSize - 1 + dy) - float64(newTy)*tileSize
			if p.nextDir != down {
				if p.nextDir == left && !lev.at(tx-1, ty).solidOrOutside() {
					p.facing = left
					p.y = float64(ty) * tileSize
					p.x = float64(tx)*tileSize - restDy
				} else if p.nextDir == right && !lev.at(tx+1, ty).solidOrOutside() {
					p.facing = right
					p.y = float64(ty) * tileSize
					p.x = float64(tx)*tileSize + restDy
				} else {
					p.y = float64(ty) * tileSize
					if !lev.at(tx, newTy).solid() {
						p.y += restDy
					}
				}
			} else {
				p.y = float64(ty) * tileSize
				if !lev.at(tx, newTy).solid() {
					p.y += restDy
				}
			}
		}
	}

	// wrap around if outside the level
	if p.x < 0 {
		p.x += float64(lev.width * tileSize)
	}
	maxX := float64(lev.width * tileSize)
	if p.x >= maxX {
		p.x -= maxX
	}
	if p.y < 0 {
		p.y += float64(lev.height * tileSize)
	}
	maxY := float64(lev.height * tileSize)
	if p.y >= maxY {
		p.y -= maxY
	}
}

func round(x float64) int {
	if x >= 0 {
		return int(x + 0.5)
	}
	return int(x - 0.5)
}

/*
	Here are all allowed level characters:

	─ │ ┌ ┐ └ ┘    walls

	· O            small and big cookie

	' '            SPACE for empty corridors

	) (            player start, facing left: ) or right: (
	               the player is moved half a tile in the facing direction so it
	               is put in the center of a two-tile wall

	?              mark areas where special fruit will appear every once in a
	               while

	/\             ghost door, it forms an arrow in the direction that the
	               ghosts come out of it

	R, P, B, G     start of the Red, Pink, Blue and Green ghosts
*/
const level1 = `
┌────────────┐┌────────────┐
│············││············│
│·┌──┐·┌───┐·││·┌───┐·┌──┐·│
│O│  │·│   │·││·│   │·│  │O│
│·└──┘·└───┘·└┘·└───┘·└──┘·│
│··························│
│·┌──┐·┌┐·┌──────┐·┌┐·┌──┐·│
│·└──┘·││·└──┐┌──┘·││·└──┘·│
│······││····││····││······│
└────┐·│└──┐ ││ ┌──┘│·┌────┘
     │·│┌──┘ └┘ └──┐│·│     
     │·││          ││·│     
     │·││ ┌──/\──┐ ││·│     
─────┘·└┘ │R   P │ └┘·└─────
      ·   │  B   │   ·      
─────┐·┌┐ │   G  │ ┌┐·┌─────
     │·││ └──────┘ ││·│     
     │·││    ??    ││·│     
     │·││ ┌──────┐ ││·│     
┌────┘·└┘ └──┐┌──┘ └┘·└────┐
│············││············│
│·┌──┐·┌───┐·││·┌───┐·┌──┐·│
│·└─┐│·└───┘·└┘·└───┘·│┌─┘·│
│O··││······· )·······││··O│
└─┐·││·┌┐·┌──────┐·┌┐·││·┌─┘
┌─┘·└┘·││·└──┐┌──┘·││·└┘·└─┐
│······││····││····││······│
│·┌────┘└──┐·││·┌──┘└────┐·│
│·└────────┘·└┘·└────────┘·│
│··························│
└──────────────────────────┘
`

const level2 = `
┌────────────┐
│············│
│·┌────────┐·│
│·│┌──────┐│·│
│·││······││·│
│·││·┌──┐·││·│
│·││·└─┐│·││·│
│·││·( ││·││·│
│·│└───┘│·││·│
│·└─────┘·││·│
│·········││·│
└─────────┘└─┘
`
