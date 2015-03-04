package main

import (
	"fmt"
	"github.com/gonutz/prototype/draw"
	"math"
)

func main() {
	var leftPanel, rightPanel *rect
	var ball *circle
	var resetAfterScore = func(ballVelocitySign float32) {
		ball = &circle{
			320, 240, 5,
			ballVelocitySign * 8.0, 0.0,
		}
		leftPanel = &rect{10, 200, 10, 80}
		rightPanel = &rect{620, 200, 10, 80}
	}
	resetAfterScore(1.0)
	leftScore := 0
	rightScore := 0

	draw.RunWindow("Tennis - press N to restart", 640, 480, func(window *draw.Window) {

		if window.WasKeyPressed("escape") {
			window.Running = false
		}

		if window.WasKeyPressed("n") {
			resetAfterScore(1.0)
			leftScore = 0
			rightScore = 0
			return
		}

		const dy = 7
		if window.IsKeyDown("down") {
			rightPanel.y += dy
		}
		if window.IsKeyDown("up") {
			rightPanel.y -= dy
		}
		if window.IsKeyDown("lctrl") {
			leftPanel.y += dy
		}
		if window.IsKeyDown("lshift") {
			leftPanel.y -= dy
		}
		keepInYBounds(leftPanel, 480)
		keepInYBounds(rightPanel, 480)

		for i := 0; i < 4; i++ {
			ball.move(0.25)
			ball.collideLeft(leftPanel)
			ball.collideRight(rightPanel)
			ball.collideWall(480)
		}

		if ball.isInLeftGoal() {
			rightScore++
			resetAfterScore(1.0)
		}
		if ball.isInRightGoal(640) {
			leftScore++
			resetAfterScore(-1.0)
		}

		window.FillRect(0, 0, 640, 480, draw.DarkGreen)
		leftPanel.draw(window)
		rightPanel.draw(window)
		ball.draw(window)
		printScore(leftScore, rightScore, window)

	})
}

type rect struct {
	x, y, w, h int
}

func (r *rect) draw(window *draw.Window) {
	window.FillRect(r.x, r.y, r.w, r.h, draw.White)
	window.DrawRect(r.x, r.y, r.w, r.h, draw.Black)
}

type circle struct {
	centerX, centerY float32
	radius           int
	vx, vy           float32
}

func (c *circle) draw(window *draw.Window) {
	x := int(c.centerX) - c.radius
	y := int(c.centerY) - c.radius
	size := 2 * c.radius
	window.FillEllipse(x, y, size, size, draw.White)
	window.DrawEllipse(x, y, size, size, draw.Black)
}

func (c *circle) move(dt float32) {
	c.centerX += c.vx * dt
	c.centerY += c.vy * dt
}

func (c *circle) collideLeft(r *rect) {
	if c.vx > 0 {
		return
	}
	c.collide(r, r.x+r.w)
}

func (c *circle) collide(r *rect, x int) {
	cx, cy := int(c.centerX), int(c.centerY)
	r2 := c.radius * c.radius
	pointInCircle := func(x, y int) bool {
		return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r2
	}
	for y := r.y; y < r.y+r.h; y++ {
		if pointInCircle(x, y) {
			c.vx = -c.vx
			c.shiftAngle(float32(y-r.y) / float32(r.h-1))
			return
		}
	}
}

func (c *circle) shiftAngle(heightPercentag float32) {
	length := math.Sqrt(float64(c.vx*c.vx + c.vy*c.vy))
	angle := float64(0.4*math.Pi - 0.8*math.Pi*heightPercentag)
	if heightPercentag >= 0.4 && heightPercentag <= 0.6 {
		angle = 0.0
	}
	dx, dy := math.Cos(angle), -math.Sin(angle)
	if c.vx < 0 {
		dx = -dx
	}
	c.vx = float32(dx * length)
	c.vy = float32(dy * length)
}

func (c *circle) collideRight(r *rect) {
	if c.vx < 0 {
		return
	}
	c.collide(r, r.x)
}

func (c *circle) collideWall(height int) {
	if c.centerY < float32(c.radius) ||
		c.centerY+float32(c.radius) >= float32(height) {
		c.vy = -c.vy
	}
}

func (c *circle) isInLeftGoal() bool {
	return c.centerX < float32(-c.radius)
}

func (c *circle) isInRightGoal(width int) bool {
	return c.centerX > float32(width+c.radius)
}

func keepInYBounds(r *rect, height int) {
	if r.y < 0 {
		r.y = 0
	}
	if r.y+r.h >= height {
		r.y = height - r.h
	}
}

func printScore(left, right int, window *draw.Window) {
	l, r := fmt.Sprintf("%v", left), fmt.Sprintf("%v", right)
	for len(l) < 5 {
		l = " " + l
	}
	for len(r) < 5 {
		r = r + " "
	}
	window.DrawScaledText(l+" : "+r, 200, 10, 2.0, draw.Black)
}
