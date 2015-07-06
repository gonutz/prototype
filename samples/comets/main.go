package main

import (
	"github.com/gonutz/prototype/draw"
	"math"
	"os"
	"path/filepath"
)

const (
	screenWidth  = 800
	screenHeight = 600
	maxSpeed     = 10
	bulletSpeed  = 13
)

func main() {
	ship := newShip()
	ship.x, ship.y = screenWidth/2, screenHeight/2
	const shootDelay = 8
	nextShot := 0
	var bullets []*bullet
	var comets []*comet

	reset := func() {
		ship = newShip()
		ship.x, ship.y = screenWidth/2, screenHeight/2
		bullets = nil
		nextShot = 0
	}

	comets = append(comets, &comet{
		angle: 0, rotation: 0.1,
		position: vec2{20, 40},
		speed:    vec2{0.5, 0.2},
		outline:  []vec2{{-5, -5}, {5, -5}, {5, 5}, {-5, 5}, {-5, -5}},
	})

	draw.RunWindow("Comets", screenWidth, screenHeight, func(window draw.Window) {
		if window.WasKeyPressed("escape") {
			window.Close()
		}
		if window.WasKeyPressed("r") {
			reset()
			return
		}

		const rotation = 0.08
		if window.IsKeyDown("left") {
			ship.angle += rotation
		}
		if window.IsKeyDown("right") {
			ship.angle -= rotation
		}
		ship.setBoosting(window.IsKeyDown("up"), window)

		if nextShot > 0 {
			nextShot--
		}
		if window.IsKeyDown("space") && nextShot == 0 {
			ship.shoot(&bullets)
			window.PlaySoundFile(getPathTo("shoot.wav"))
			nextShot = shootDelay
		}

		ship.move()

		ship.draw(window)
		for _, b := range bullets {
			b.move()
			b.draw(window)
		}
		for _, c := range comets {
			c.update()
			c.draw(window)
		}
	})
}

type ship struct {
	x, y          float64
	angle         float64
	speed         vec2
	outline       []vec2
	boostOutlines [][]vec2
	color         draw.Color
	boosting      bool
	cannons       []vec2
}

type vec2 struct{ x, y float64 }

func (v vec2) plus(v2 vec2) vec2     { return vec2{v.x + v2.x, v.y + v2.y} }
func (v vec2) times(s float64) vec2  { return vec2{v.x * s, v.y * s} }
func (v vec2) squareLength() float64 { return v.x*v.x + v.y*v.y }

type screenPoint struct{ x, y int }

func newShip() *ship {
	// outline as drawn on a piece of paper, coordinates 0,0 are the ship's center
	outline := []vec2{
		{-3, -5}, {-3, -1}, {-2, 0}, {-1, 0}, {-1, 2}, {-3, 2}, {-3, 4},
		{-2, 4}, {-2, 3}, {-1, 3}, {-1, 4}, {0, 6}, {1, 4}, {1, 3}, {2, 3}, {2, 4},
		{3, 4}, {3, 2}, {1, 2}, {1, 0}, {2, 0}, {3, -1}, {3, -5}, {1, -5},
		{1, -3}, {0, -2}, {-1, -3}, {-1, -5}, {-3, -5},
	}
	boosts := [][]vec2{
		{{-3, -5}, {-2.75, -6}, {-2.5, -5.5}, {-2, -7}, {-1.5, -5.5}, {-1.25, -6}, {-1, -5}},
		{{3, -5}, {2.75, -6}, {2.5, -5.5}, {2, -7}, {1.5, -5.5}, {1.25, -6}, {1, -5}},
	}
	cannons := []vec2{{-2.5, 4}, {2.5, 4}}
	// scale everything up and invert the y-coordinates, screen has 0,0 in top-left corner
	const scale = 4
	for i := range outline {
		outline[i].x *= scale
		outline[i].y *= -scale
	}
	for i := range cannons {
		cannons[i].x *= scale
		cannons[i].y *= -scale
	}
	for i := range boosts {
		for j := range boosts[i] {
			boosts[i][j].x *= scale
			boosts[i][j].y *= -scale
		}
	}
	return &ship{
		outline:       outline,
		boostOutlines: boosts,
		color:         draw.White,
		cannons:       cannons,
	}
}

func (s *ship) draw(window draw.Window) {
	screenPoints := s.outlineToScreen()
	for i := 1; i < len(screenPoints); i++ {
		a, b := screenPoints[i-1], screenPoints[i]
		window.DrawLine(a.x, a.y, b.x, b.y, s.color)
	}
	if s.boosting {
		boosts := s.boostOutlinesToScreen()
		for _, boost := range boosts {
			for i := 1; i < len(boost); i++ {
				a, b := boost[i-1], boost[i]
				window.DrawLine(a.x, a.y, b.x, b.y, s.color)
			}
		}
	}
}

func (s *ship) outlineToScreen() []screenPoint {
	return s.pointsToScreen(s.outline)
}

func (s *ship) pointsToScreen(points []vec2) []screenPoint {
	screenPoints := make([]screenPoint, len(points))
	sin, cos := math.Sincos(s.angle)
	for i, p := range points {
		x, y := float64(p.x), float64(p.y)
		x, y = s.x+cos*x+sin*y, s.y-sin*x+cos*y
		screenPoints[i] = screenPoint{
			int(x + 0.5), int(y + 0.5),
		}
	}
	return screenPoints
}

func (s *ship) boostOutlinesToScreen() [][]screenPoint {
	boosts := make([][]screenPoint, len(s.boostOutlines))
	for i := range boosts {
		boosts[i] = s.pointsToScreen(s.boostOutlines[i])
	}
	return boosts
}

func (s *ship) direction() vec2 {
	dy, dx := math.Sincos(-math.Pi/2 - s.angle)
	return vec2{dx, dy}
}

func (s *ship) setBoosting(boosting bool, window draw.Window) {
	if !s.boosting && boosting {
		window.PlaySoundFile(getPathTo("boost.wav"))
	}
	s.boosting = boosting
}

func getPathTo(local string) string {
	gopath := os.Getenv("GOPATH")
	return filepath.Join(gopath, "src", "github.com", "gonutz", "prototype", "samples", "comets", local)
}

func (s *ship) move() {
	if s.boosting {
		dir := s.direction().times(0.1)
		s.speed = s.speed.plus(dir)
		lSquare := s.speed.squareLength()
		if lSquare > maxSpeed*maxSpeed {
			scale := maxSpeed / math.Sqrt(lSquare)
			s.speed.x *= scale
			s.speed.y *= scale
		}
	}
	s.x += s.speed.x
	s.y += s.speed.y

	const safety = 10
	for s.x < -safety {
		s.x += screenWidth + 2*safety
	}
	for s.y < -safety {
		s.y += screenHeight + 2*safety
	}
	for s.x >= screenWidth+2*safety {
		s.x -= screenWidth + 2*safety
	}
	for s.y >= screenHeight+2*safety {
		s.y -= screenHeight + 2*safety
	}
}

func (s *ship) shoot(bullets *[]*bullet) {
	cannons := s.pointsToScreen(s.cannons)
	for _, cannon := range cannons {
		*bullets = append(*bullets, &bullet{
			position: vec2{float64(cannon.x), float64(cannon.y)},
			speed:    s.direction().times(bulletSpeed),
		})
	}
}

type bullet struct {
	position vec2
	speed    vec2
}

func (b *bullet) move() {
	b.position = b.position.plus(b.speed)
}

func (b *bullet) draw(window draw.Window) {
	window.DrawPoint(int(b.position.x+0.5), int(b.position.y+0.5), draw.White)
}

type comet struct {
	angle, rotation float64
	position        vec2
	speed           vec2
	outline         []vec2
}

func (c *comet) update() {
	c.angle += c.rotation
	c.position = c.position.plus(c.speed)
}

func (c *comet) draw(window draw.Window) {
	outline := c.toScreenPoints(c.outline)
	for i := 1; i < len(outline); i++ {
		window.DrawLine(outline[i-1].x, outline[i-1].y, outline[i].x, outline[i].y, draw.White)
	}
}

func (c *comet) toScreenPoints(ps []vec2) []screenPoint {
	screenPoints := make([]screenPoint, len(ps))
	sin, cos := math.Sincos(c.angle)
	for i := range ps {
		x, y := cos*ps[i].x-sin*ps[i].y, sin*ps[i].x+cos*ps[i].y
		screenPoints[i] = screenPoint{
			int(c.position.x + x + 0.5),
			int(c.position.y + y + 0.5),
		}
	}
	return screenPoints
}
