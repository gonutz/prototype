package main

import "github.com/gonutz/prototype/draw"

const (
	windowWidth        = 640
	windowHeight       = 480
	shipSize           = 20
	shipSpeed          = 5
	bulletDelay        = 10
	patternChangeDelay = 20
	enemySize          = 30
	bulletSize         = 8
	bulletSpeed        = 8
)

func main() {
	var (
		ship              spaceShip
		bullets           []bullet
		nextBulletLock    int // Only shoot at bulletDelay intervals.
		enemies           []enemy
		enemyMovePattern  int
		nextPatternChange int
		gameOver          bool
		gameOverText      string

		// Enemies move left, down, right, down, left, down, ...
		movePatterns = []struct{ dx, dy int }{
			{-1, 0}, {-1, 0},
			{0, 1},
			{1, 0}, {1, 0}, {1, 0}, {1, 0},
			{0, 1},
			{-1, 0}, {-1, 0},
		}
	)

	newGame := func() {
		ship = spaceShip{
			x: (windowWidth - shipSize) / 2,
			y: windowHeight - 10 - shipSize,
		}
		bullets = nil
		nextBulletLock = 0
		enemies = createEnemies()
		enemyMovePattern = 0
		nextPatternChange = patternChangeDelay
		gameOver = false
		gameOverText = ""
	}
	newGame()

	err := draw.RunWindow("Space Shooter", windowWidth, windowHeight, func(window draw.Window) {
		if window.WasKeyPressed(draw.KeyEscape) {
			window.Close()
		}
		if window.WasKeyPressed(draw.KeyN) {
			newGame()
		}

		if !gameOver {

			if window.IsKeyDown(draw.KeyLeft) {
				ship.move(-shipSpeed)
			}
			if window.IsKeyDown(draw.KeyRight) {
				ship.move(shipSpeed)
			}
			nextBulletLock--
			if window.IsKeyDown(draw.KeySpace) || window.WasKeyPressed(draw.KeySpace) {
				if nextBulletLock <= 0 {
					bullets = append(
						bullets,
						bullet{x: ship.x + shipSize/2, y: ship.y + shipSize/2},
					)
					nextBulletLock = bulletDelay
					window.PlaySoundFile("shoot.wav")
				}
			}

			for i := range bullets {
				bullets[i].y -= bulletSpeed
				for j := range enemies {
					bullets[i].collide(&enemies[j], window)
				}
			}
			bullets = removeDeadBullets(bullets)

			nextPatternChange--
			if nextPatternChange <= 0 {
				nextPatternChange = patternChangeDelay
				enemyMovePattern = (enemyMovePattern + 1) % len(movePatterns)
			}
			p := movePatterns[enemyMovePattern]
			for i := range enemies {
				enemies[i].move(p.dx, p.dy)
			}
			enemies = removeDeadEnemies(enemies)
			if len(enemies) == 0 {
				gameOver = true
				gameOverText = "You defeated the enemy!\n  Press n to restart."
			}
			for _, e := range enemies {
				if e.y >= ship.y {
					gameOver = true
					gameOverText = " You lost the battle...\n   Press n to restart"
				}
			}
		}

		window.FillEllipse(ship.x, ship.y, shipSize, shipSize, draw.Red)
		for _, b := range bullets {
			window.FillEllipse(b.x-bulletSize/2, b.y-bulletSize/2, bulletSize, bulletSize, draw.Green)
		}
		for _, e := range enemies {
			window.FillRect(e.x-enemySize/2, e.y-enemySize/2, enemySize, enemySize, e.color)
		}
		window.DrawScaledText(gameOverText, 100, 180, 2.0, draw.White)
	})

	if err != nil {
		panic(err)
	}
}

func createEnemies() []enemy {
	var enemies []enemy
	colors := []draw.Color{draw.DarkBlue, draw.Blue, draw.LightBlue}
	for y := 0; y < 3; y++ {
		for x := 70; x <= 640-50-enemySize/2; x += 25 + enemySize {
			enemies = append(enemies, enemy{
				x:     x,
				y:     30 + y*(enemySize+10),
				life:  3 - y,
				color: colors[y],
			})
		}
	}
	return enemies
}

type spaceShip struct {
	x, y int // (x,y) is the top-left of the square ship.
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
	x, y int // (x,y) is the center of the circular bullet.
	dead bool
}

func (b *bullet) collide(e *enemy, window draw.Window) {
	squareDist := square(b.x-e.x) + square(b.y-e.y)
	if squareDist <= square(enemySize/2+bulletSize/2) {
		e.life--
		b.dead = true
		if e.life == 0 {
			window.PlaySoundFile("explosion.wav")
		} else {
			window.PlaySoundFile("hit.wav")
		}
	}
}

func square(x int) int {
	return x * x
}

func removeDeadBullets(bullets []bullet) []bullet {
	alive := bullets[:0] // Reuse the array.
	for _, b := range bullets {
		if b.y > -100 && !b.dead {
			alive = append(alive, b)
		}
	}
	return alive
}

type enemy struct {
	x, y  int
	life  int
	color draw.Color
}

func (e *enemy) move(dx, dy int) {
	e.x += dx
	e.y += dy
}

func removeDeadEnemies(enemies []enemy) []enemy {
	alive := enemies[:0] // Reuse the array.
	for _, e := range enemies {
		if e.life > 0 {
			alive = append(alive, e)
		}
	}
	return alive
}
