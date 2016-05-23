package main

import (
	"fmt"
	"github.com/gonutz/prototype/draw"
	"math/rand"
	"time"
)

const tileSize = 20
const mapSize = 13
const windowSize = tileSize * mapSize

func main() {
	var frame int
	var theSnake *snake
	var cookie point
	var resetCookie func()
	resetCookie = func() {
		cookie = point{rand.Intn(mapSize), rand.Intn(mapSize)}
		for _, p := range theSnake.body {
			if p == cookie {
				resetCookie()
			}
		}
	}
	var score int
	var moveDelay int
	const minMoveDelay = 5
	var nextMove int
	var gameOver bool
	resetGame := func() {
		rand.Seed(time.Now().UnixNano())
		frame = 0
		center := mapSize / 2
		theSnake = &snake{
			body: []point{
				{center - 1, center},
				{center, center},
				{center + 1, center},
			}}
		resetCookie()
		score = 0
		moveDelay = 10
		nextMove = moveDelay
		gameOver = false
	}

	resetGame()

	mainErr := draw.RunWindow("Eat everything", windowSize, windowSize+tileSize,
		func(window draw.Window) {

			if window.WasKeyPressed(draw.KeyEscape) {
				window.Close()
			}
			if window.WasKeyPressed(draw.KeyEnter) {
				resetGame()
				return
			}

			if !gameOver {
				if window.WasKeyPressed(draw.KeyLeft) {
					theSnake.setVelocity(-1, 0)
				}
				if window.WasKeyPressed(draw.KeyRight) {
					theSnake.setVelocity(1, 0)
				}
				if window.WasKeyPressed(draw.KeyDown) {
					theSnake.setVelocity(0, 1)
				}
				if window.WasKeyPressed(draw.KeyUp) {
					theSnake.setVelocity(0, -1)
				}

				nextMove--
				if nextMove <= 0 {
					theSnake.move(cookie)
					nextMove = moveDelay
				}

				if theSnake.head() == cookie {
					resetCookie()
					score++
					if score%10 == 0 {
						moveDelay--
					}
					if moveDelay < minMoveDelay {
						moveDelay = minMoveDelay
					}
				}

				if theSnake.bitItself() {
					gameOver = true
				}
				frame++
			}

			window.FillRect(0, 0, windowSize, windowSize, draw.LightGreen)
			theSnake.draw(window, frame)
			drawCookie(cookie, window)
			drawScore(window, score)
			if gameOver {
				window.DrawScaledText(" Game Over!\npress  ENTER\n to restart", 25, 80, 2.0, draw.White)
			}

		})

	if mainErr != nil {
		panic(mainErr)
	}
}

type point struct{ x, y int }

type snake struct {
	body   []point
	dx, dy int
}

func (s *snake) drawBody(window draw.Window) {
	for _, p := range s.body {
		x := p.x * tileSize
		y := p.y * tileSize
		window.FillEllipse(x, y, tileSize, tileSize, draw.DarkGreen)
		window.DrawEllipse(x, y, tileSize, tileSize, draw.Black)
	}
}

func (s *snake) drawClaws(window draw.Window, frame int) {
	head := s.head()
	x := head.x * tileSize
	y := head.y * tileSize
	offset := 0
	if (frame/8)%2 == 0 {
		offset = 2
	}
	if s.dy < 0 {
		window.FillEllipse(x+offset, y, 2, 4, draw.Red)
		window.FillEllipse(x+tileSize-2-offset, y, 2, 4, draw.Red)
	} else if s.dy > 0 {
		window.FillEllipse(x+offset, y+tileSize-3, 2, 4, draw.Red)
		window.FillEllipse(x+tileSize-2-offset, y+tileSize-3, 2, 4, draw.Red)
	} else if s.dx > 0 {
		window.FillEllipse(x+tileSize-3, y+offset, 4, 2, draw.Red)
		window.FillEllipse(x+tileSize-3, y+tileSize-2-offset, 4, 2, draw.Red)
	} else {
		window.FillEllipse(x, y+offset, 4, 2, draw.Red)
		window.FillEllipse(x, y+tileSize-2-offset, 4, 2, draw.Red)
	}
}

func (s *snake) drawEyes(window draw.Window) {
	head := s.head()
	x := head.x * tileSize
	y := head.y * tileSize
	if s.dy < 0 {
		window.FillEllipse(x+5, y+5, 2, 2, draw.Black)
		window.FillEllipse(x+tileSize-8, y+5, 2, 2, draw.Black)
	} else if s.dy > 0 {
		window.FillEllipse(x+5, y+tileSize-7, 2, 2, draw.Black)
		window.FillEllipse(x+tileSize-8, y+tileSize-7, 2, 2, draw.Black)
	} else if s.dx > 0 {
		window.FillEllipse(x+tileSize-6, y+6, 2, 2, draw.Black)
		window.FillEllipse(x+tileSize-6, y+tileSize-7, 2, 2, draw.Black)
	} else {
		window.FillEllipse(x+5, y+6, 2, 2, draw.Black)
		window.FillEllipse(x+5, y+tileSize-7, 2, 2, draw.Black)
	}
}

func (s *snake) draw(window draw.Window, frame int) {
	s.drawBody(window)
	s.drawClaws(window, frame)
	s.drawEyes(window)
}

func (s *snake) setVelocity(dx, dy int) {
	newHead := s.nextHeadPosition(dx, dy)
	if newHead != s.body[1] {
		s.dx = dx
		s.dy = dy
	}
}

func (s *snake) nextHeadPosition(dx, dy int) point {
	newHead := s.head()
	newHead.x = (newHead.x + dx + mapSize) % mapSize
	newHead.y = (newHead.y + dy + mapSize) % mapSize
	return newHead
}

func (s *snake) move(cookie point) {
	if s.dx == 0 && s.dy == 0 {
		return
	}
	newHead := s.nextHeadPosition(s.dx, s.dy)
	tail := s.body[:len(s.body)-1]
	if newHead == cookie {
		tail = s.body
	}
	s.body = append([]point{newHead}, tail...)
}

func (s *snake) head() point {
	return s.body[0]
}

func (s *snake) bitItself() bool {
	head := s.head()
	for _, p := range s.body[1:] {
		if p == head {
			return true
		}
	}
	return false
}

func drawCookie(cookie point, window draw.Window) {
	const margin = 4
	x := cookie.x*tileSize + margin
	y := cookie.y*tileSize + margin
	size := tileSize - 2*margin
	window.FillEllipse(x, y, size, size, draw.DarkYellow)
	window.DrawEllipse(x, y, size, size, draw.Black)
	window.DrawPoint(x+4, y+5, draw.Black)
	window.DrawPoint(x+9, y+6, draw.Black)
	window.DrawPoint(x+7, y+8, draw.Black)
}

func drawScore(window draw.Window, score int) {
	cookieText := "cookies"
	if score == 1 {
		cookieText = "cookie"
	}
	window.DrawText(fmt.Sprintf("You ate %v %v", score, cookieText),
		10, windowSize+2, draw.White)
}
