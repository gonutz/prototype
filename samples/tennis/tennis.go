package main

import (
	"flag"
	"fmt"
	"github.com/gonutz/prototype/draw"
	"math"
	"os"
	"path/filepath"
)

func main() {
	speed := flag.Float64("speed", 1.0, "The regular speed is multiplied by this factor")
	flag.Parse()
	if *speed < 0.01 {
		*speed = 0.01
	}
	if *speed > 100 {
		*speed = 100
	}

	var leftPanel, rightPanel *rect
	var ball *circle
	var resetAfterScore = func(ballVelocitySign float32) {
		ball = &circle{
			320, 240, 5,
			float32(*speed) * ballVelocitySign * 8.0, 0.0,
		}
		leftPanel = &rect{10, 200, 10, 80}
		rightPanel = &rect{620, 200, 10, 80}
	}
	resetAfterScore(1.0)
	leftScore := 0
	rightScore := 0
	samplesPath := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "gonutz", "prototype", "samples")
	scoreSound := filepath.Join(samplesPath, "tennis", "score.wav")
	panelBounceSound := filepath.Join(samplesPath, "tennis", "bounce.wav")
	wallBounceSound := filepath.Join(samplesPath, "tennis", "bounce2.wav")

	mainErr := draw.RunWindow("Tennis - press N to restart", 640, 480,
		func(window draw.Window) {

			if window.WasKeyPressed(draw.KeyEscape) {
				window.Close()
			}

			if window.WasKeyPressed(draw.KeyN) {
				resetAfterScore(1.0)
				leftScore = 0
				rightScore = 0
				return
			}

			const dy = 7
			if window.IsKeyDown(draw.KeyDown) {
				rightPanel.y += dy
			}
			if window.IsKeyDown(draw.KeyUp) {
				rightPanel.y -= dy
			}
			if window.IsKeyDown(draw.KeyLeftControl) {
				leftPanel.y += dy
			}
			if window.IsKeyDown(draw.KeyLeftShift) {
				leftPanel.y -= dy
			}
			keepInYBounds(leftPanel, 480)
			keepInYBounds(rightPanel, 480)

			timeDivision := 4
			if *speed > 1.0 {
				timeDivision = int(float64(timeDivision) * *speed)
			}
			for i := 0; i < timeDivision; i++ {
				ball.move(1.0 / float32(timeDivision))
				if ball.collideLeft(leftPanel) || ball.collideRight(rightPanel) {
					window.PlaySoundFile(panelBounceSound)
				}
				if ball.collideWall(480) {
					window.PlaySoundFile(wallBounceSound)
				}
			}

			if ball.isInLeftGoal() {
				rightScore++
				window.PlaySoundFile(scoreSound)
				resetAfterScore(1.0)
			}
			if ball.isInRightGoal(640) {
				leftScore++
				window.PlaySoundFile(scoreSound)
				resetAfterScore(-1.0)
			}

			window.FillRect(0, 0, 640, 480, draw.DarkGreen)
			window.DrawRect(50, 50, 540, 380, draw.LightGreen)
			window.DrawLine(320, 50, 320, 429, draw.LightGreen)
			leftPanel.draw(window)
			rightPanel.draw(window)
			ball.draw(window)
			drawScore(leftScore, rightScore, window)

		})

	if mainErr != nil {
		panic(mainErr)
	}
}

type rect struct {
	x, y, w, h int
}

func (r *rect) draw(window draw.Window) {
	window.FillRect(r.x, r.y, r.w, r.h, draw.White)
	window.DrawRect(r.x, r.y, r.w, r.h, draw.Black)
}

type circle struct {
	centerX, centerY float32
	radius           int
	vx, vy           float32
}

func (c *circle) draw(window draw.Window) {
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

func (c *circle) collideLeft(r *rect) bool {
	if c.vx > 0 {
		return false
	}
	return c.collide(r, r.x+r.w)
}

func (c *circle) collide(r *rect, x int) bool {
	cx, cy := int(c.centerX), int(c.centerY)
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

func (c *circle) collideRight(r *rect) bool {
	if c.vx < 0 {
		return false
	}
	return c.collide(r, r.x)
}

func (c *circle) collideWall(height int) bool {
	if c.centerY < float32(c.radius) ||
		c.centerY+float32(c.radius) >= float32(height) {
		c.vy = -c.vy
		return true
	}
	return false
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

func drawScore(left, right int, window draw.Window) {
	l, r := fmt.Sprintf("%v", left), fmt.Sprintf("%v", right)
	for len(l) < 5 {
		l = " " + l
	}
	for len(r) < 5 {
		r = r + " "
	}
	scoreText := l + " : " + r
	const scale = 2.0
	w, _ := window.GetScaledTextSize(scoreText, scale)
	window.DrawScaledText(scoreText, 320-w/2+1, 10, scale, draw.Black)
}
