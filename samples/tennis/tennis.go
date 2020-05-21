package main

import (
	"math"
	"strconv"

	"github.com/gonutz/prototype/draw"
)

const (
	windowWidth, windowHeight = 640, 480
	ballSpeed                 = 8
	panelSpeed                = 7
	ballRadius                = 5
)

func main() {
	var (
		leftPanel  rect
		rightPanel rect
		ball       circle
		leftScore  int
		rightScore int
	)

	resetAfterScore := func(ballVelocitySign float32) {
		ball = circle{
			centerX: windowWidth / 2,
			centerY: windowHeight / 2,
			radius:  ballRadius,
			vx:      ballVelocitySign * ballSpeed,
			vy:      0.0,
		}
		leftPanel = rect{10, windowHeight/2 - 40, 10, 80}
		rightPanel = rect{windowWidth - 20, windowHeight/2 - 40, 10, 80}
	}
	resetAfterScore(1.0)

	const title = "Tennis - press N to restart"
	err := draw.RunWindow(title, windowWidth, windowHeight, func(window draw.Window) {
		if window.WasKeyPressed(draw.KeyEscape) {
			window.Close()
		}

		if window.WasKeyPressed(draw.KeyN) {
			resetAfterScore(1.0)
			leftScore = 0
			rightScore = 0
			return
		}

		if window.IsKeyDown(draw.KeyDown) {
			rightPanel.y += panelSpeed
		}
		if window.IsKeyDown(draw.KeyUp) {
			rightPanel.y -= panelSpeed
		}
		if window.IsKeyDown(draw.KeyLeftControl) {
			leftPanel.y += panelSpeed
		}
		if window.IsKeyDown(draw.KeyLeftShift) {
			leftPanel.y -= panelSpeed
		}
		leftPanel.clampInY()
		rightPanel.clampInY()

		// Move the ball not all at once, split it and check for collisions
		// each time. This way we will not warp through obstacles.
		timeDivision := ballSpeed / 10
		if timeDivision == 0 {
			timeDivision = 1
		}
		for i := 0; i < timeDivision; i++ {
			ball.move(1.0 / float32(timeDivision))
			if ball.collideLeft(&leftPanel) || ball.collideRight(&rightPanel) {
				window.PlaySoundFile("bounce.wav")
			}
			if ball.collideWall() {
				window.PlaySoundFile("bounce2.wav")
			}
		}

		if ball.isInLeftGoal() {
			rightScore++
			window.PlaySoundFile("score.wav")
			resetAfterScore(1.0)
		}
		if ball.isInRightGoal() {
			leftScore++
			window.PlaySoundFile("score.wav")
			resetAfterScore(-1.0)
		}

		window.FillRect(0, 0, windowWidth, windowHeight, draw.DarkGreen)
		window.DrawRect(50, 50, windowWidth-100, windowHeight-100, draw.LightGreen)
		window.DrawLine(windowWidth/2, 50, windowWidth/2, windowHeight-50, draw.LightGreen)
		leftPanel.draw(window)
		rightPanel.draw(window)
		ball.draw(window)
		drawScore(leftScore, rightScore, window)
	})

	if err != nil {
		panic(err)
	}
}

type rect struct {
	x, y, w, h int
}

func (r rect) draw(window draw.Window) {
	window.FillRect(r.x, r.y, r.w, r.h, draw.White)
	window.DrawRect(r.x, r.y, r.w, r.h, draw.Black)
}

func (r *rect) clampInY() {
	if r.y < 0 {
		r.y = 0
	}
	if r.y+r.h >= windowHeight {
		r.y = windowHeight - r.h
	}
}

type circle struct {
	centerX, centerY float32
	radius           int
	vx, vy           float32
}

func (c *circle) draw(window draw.Window) {
	x := int(c.centerX+0.5) - c.radius
	y := int(c.centerY+0.5) - c.radius
	size := 2 * c.radius
	window.FillEllipse(x, y, size, size, draw.White)
	window.DrawEllipse(x, y, size, size, draw.Black)
}

func (c *circle) move(dt float32) {
	c.centerX += c.vx * dt
	c.centerY += c.vy * dt
}

func (c *circle) collideLeft(r *rect) bool {
	return c.vx < 0 && c.collide(r, r.x+r.w)
}

func (c *circle) collideRight(r *rect) bool {
	return c.vx > 0 && c.collide(r, r.x)
}

func (c *circle) collide(r *rect, x int) bool {
	cx, cy := int(c.centerX+0.5), int(c.centerY+0.5)
	r2 := c.radius * c.radius
	pointInCircle := func(x, y int) bool {
		return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r2
	}
	for y := r.y; y < r.y+r.h; y++ {
		if pointInCircle(x, y) {
			c.vx = -c.vx
			c.shiftAngle(float32(y-r.y) / float32(r.h-1))
			return true
		}
	}
	return false
}

func (c *circle) shiftAngle(heightPercent float32) {
	length := math.Sqrt(float64(c.vx*c.vx + c.vy*c.vy))
	angle := float64(0.4*math.Pi - 0.8*math.Pi*heightPercent)
	if heightPercent >= 0.4 && heightPercent <= 0.6 {
		angle = 0.0
	}
	dx, dy := math.Cos(angle), -math.Sin(angle)
	if c.vx < 0 {
		dx = -dx
	}
	c.vx = float32(dx * length)
	c.vy = float32(dy * length)
}

func (c *circle) collideWall() bool {
	if c.centerY < float32(c.radius) ||
		c.centerY+float32(c.radius) >= windowHeight {
		c.vy = -c.vy
		return true
	}
	return false
}

func (c *circle) isInLeftGoal() bool {
	return c.centerX < float32(-c.radius)
}

func (c *circle) isInRightGoal() bool {
	return c.centerX > float32(windowWidth+c.radius)
}

func drawScore(left, right int, window draw.Window) {
	// Pad scores with spaces to make it center align right.
	l, r := strconv.Itoa(left), strconv.Itoa(right)
	for len(l) < 5 {
		l = " " + l
	}
	for len(r) < 5 {
		r = r + " "
	}
	scoreText := l + " : " + r
	const scale = 2.0
	w, _ := window.GetScaledTextSize(scoreText, scale)
	window.DrawScaledText(scoreText, (windowWidth-w)/2+1, 10, scale, draw.Black)
}
