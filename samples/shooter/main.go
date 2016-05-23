package main

import (
	"github.com/gonutz/prototype/draw"
	"math"
	"os"
	"path/filepath"
)

func main() {
	ship := &spaceShip{310, 450}
	const speed = 5
	var bullets []*bullet
	nextBullet := 0
	const bulletDelay = 10
	enemies := spreadEnemies()
	movePatterns := []struct{ dx, dy int }{
		{-1, 0}, {-1, 0},
		{0, 1},
		{1, 0}, {1, 0}, {1, 0}, {1, 0},
		{0, 1},
		{-1, 0}, {-1, 0},
	}
	pattern := 0
	const patternChangeDelay = 20
	nextPatternChange := patternChangeDelay
	gameOver := false
	gameOverText := ""

	reset := func() {
		ship = &spaceShip{310, 450}
		bullets = nil
		nextBullet = 0
		enemies = spreadEnemies()
		pattern = 0
		nextPatternChange = patternChangeDelay
		gameOver = false
		gameOverText = ""
	}

	mainErr := draw.RunWindow("Space Shooter", 640, 480, func(window draw.Window) {
		if window.WasKeyPressed(draw.KeyEscape) {
			window.Close()
		}
		if window.WasKeyPressed(draw.KeyN) {
			reset()
		}

		if !gameOver {
			if window.IsKeyDown(draw.KeyLeft) {
				ship.move(-speed)
			}
			if window.IsKeyDown(draw.KeyRight) {
				ship.move(speed)
			}
			if window.IsKeyDown(draw.KeySpace) || window.WasKeyPressed(draw.KeySpace) {
				if nextBullet <= 0 {
					bullets = append(bullets,
						&bullet{x: ship.x + 10, y: ship.y - 5})
					nextBullet = bulletDelay
					window.PlaySoundFile(resourcePath("shoot.wav"))
				}
			}
			nextBullet--
			nextPatternChange--

			if nextPatternChange <= 0 {
				nextPatternChange = patternChangeDelay
				pattern = (pattern + 1) % len(movePatterns)
			}
			p := movePatterns[pattern]
			for _, e := range enemies {
				e.move(p.dx, p.dy)
			}
			for _, b := range bullets {
				b.move(-8)
				for _, e := range enemies {
					b.collide(e, window)
				}
			}
			bullets = removeDeadBullets(bullets)
			enemies = removeDeadEnemies(enemies)

			if len(enemies) == 0 {
				gameOver = true
				gameOverText = "You defeated the enemy!\n  Press n to restart."
			}
			for _, e := range enemies {
				if e.y >= 450 {
					gameOver = true
					gameOverText = " You lost the battle...\n   Press n to restart"
				}
			}
		}

		ship.draw(window)
		for _, b := range bullets {
			b.draw(window)
		}
		for _, e := range enemies {
			e.draw(window)
		}
		window.DrawScaledText(gameOverText, 100, 180, 2.0, draw.White)
	})

	if mainErr != nil {
		panic(mainErr)
	}
}

func resourcePath(filename string) string {
	projectPath := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "gonutz", "prototype", "samples", "shooter")
	return filepath.Join(projectPath, filename)
}

func spreadEnemies() []*enemy {
	enemies := make([]*enemy, 0, 50)
	const size = 30
	colors := []draw.Color{draw.DarkBlue, draw.Blue, draw.LightBlue}
	for y := 0; y < 3; y++ {
		for x := 70; x <= 640-50-size/2; x += 25 + size {
			enemies = append(enemies, &enemy{
				x:     x,
				y:     30 + y*(size+10),
				size:  size,
				life:  3 - y,
				color: colors[y],
			})
		}
	}
	return enemies
}

type spaceShip struct {
	x, y int
}

func (s *spaceShip) draw(window draw.Window) {
	window.FillEllipse(s.x, s.y, 20, 20, draw.Red)
}

func (s *spaceShip) move(dx int) {
	s.x += dx
	if s.x < 0 {
		s.x = 0
	}
	if s.x > 620 {
		s.x = 620
	}
}

type bullet struct {
	x, y int
	dead bool
}

const bulletSize = 8

func (b *bullet) draw(window draw.Window) {
	window.FillEllipse(b.x-bulletSize/2, b.y-bulletSize/2, bulletSize, bulletSize, draw.Green)
}

func (b *bullet) move(dy int) {
	b.y += dy
}

func (b *bullet) collide(e *enemy, window draw.Window) {
	dx := abs(b.x - e.x)
	dy := abs(b.y - e.y)
	dist := int(math.Sqrt(float64(dx*dx+dy*dy)) + 0.5)
	if dist <= e.size/2+bulletSize/2 {
		e.life--
		b.dead = true
		if e.life == 0 {
			window.PlaySoundFile(resourcePath("explosion.wav"))
		} else {
			window.PlaySoundFile(resourcePath("hit.wav"))
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func removeDeadBullets(bullets []*bullet) []*bullet {
	inBounds := make([]*bullet, 0, len(bullets))
	for _, b := range bullets {
		if b.y > -100 && !b.dead {
			inBounds = append(inBounds, b)
		}
	}
	return inBounds
}

type enemy struct {
	x, y, size int
	life       int
	color      draw.Color
}

func (e *enemy) draw(window draw.Window) {
	window.FillRect(e.x-e.size/2, e.y-e.size/2, e.size, e.size, e.color)
}

func (e *enemy) move(dx, dy int) {
	e.x += dx
	e.y += dy
}

func removeDeadEnemies(enemies []*enemy) []*enemy {
	alive := make([]*enemy, 0, len(enemies))
	for _, e := range enemies {
		if e.life > 0 {
			alive = append(alive, e)
		}
	}
	return alive
}
