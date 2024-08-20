package main

import (
	"math"

	"github.com/gonutz/prototype/draw"
)

const (
	boardWidth   = 7
	boardHeight  = 6
	tileSize     = 70
	margin       = 5
	windowWidth  = margin + boardWidth*(tileSize+margin)
	windowHeight = margin + boardHeight*(tileSize+margin)
	textScale    = 2
)

const (
	empty = iota
	red
	blue
	both
)

func main() {
	var board [boardWidth][boardHeight]int
	nextPlayer := red
	time := 0.0

	isFloor := func(x, y int) bool {
		for i := 0; i <= y; i++ {
			if board[x][i] != empty {
				return false
			}
		}

		for i := y + 1; i < boardHeight; i++ {
			if board[x][i] == empty {
				return false
			}
		}

		return true
	}

	findWinner := func() int {
		freeCount := 0
		for x := 0; x < boardWidth; x++ {
			for y := 0; y <= boardHeight-4; y++ {
				if board[x][y] == empty {
					freeCount++
				}
			}
		}
		if freeCount == 0 {
			return both
		}

		// Check vertical fours.
		for x := 0; x < boardWidth; x++ {
			for y := 0; y <= boardHeight-4; y++ {
				full := board[x][y] != empty
				for i := 1; i < 4; i++ {
					full = full && board[x][y+i] == board[x][y+i-1]
				}
				if full {
					return board[x][y]
				}
			}
		}

		// Check horizontal fours.
		for y := 0; y < boardHeight; y++ {
			for x := 0; x <= boardWidth-4; x++ {
				full := board[x][y] != empty
				for i := 1; i < 4; i++ {
					full = full && board[x+i][y] == board[x+i-1][y]
				}
				if full {
					return board[x][y]
				}
			}
		}

		// Check diagonal fours.
		for x := 0; x <= boardWidth-4; x++ {
			for y := 0; y <= boardHeight-4; y++ {
				// Check top-left to bottom-right diagonals.
				full := board[x][y] != empty
				for i := 1; i < 4; i++ {
					full = full && board[x+i][y+i] == board[x+i-1][y+i-1]
				}
				if full {
					return board[x][y]
				}

				// Check top-right to bottom-left diagonals.
				full = board[x+3][y] != empty
				for i := 1; i < 4; i++ {
					full = full && board[x+3-i][y+i] == board[x+3-i+1][y+i-1]
				}
				if full {
					return board[x+3][y]
				}
			}
		}

		return empty
	}

	winner := empty

	newGame := func() {
		board = [boardWidth][boardHeight]int{}
		nextPlayer = red
		time = 0.0
		winner = empty
	}

	draw.RunWindow("Rows of Four", windowWidth, windowHeight, func(window draw.Window) {
		if window.WasKeyPressed(draw.KeyEscape) {
			window.Close()
		}

		time += 1.0 / 60

		mouseX, _ := window.MousePosition()
		mouseColumn := (mouseX - margin) / (tileSize + margin)

		if winner != empty {
			for _, c := range window.Clicks() {
				if c.Button == draw.RightButton {
					newGame()
				}
			}
		} else {
			for _, c := range window.Clicks() {
				if c.Button == draw.LeftButton {
					x := mouseColumn
					for y := boardHeight - 1; y >= 0; y-- {
						if board[x][y] == empty {
							board[x][y] = nextPlayer

							winner = findWinner()

							if nextPlayer == red {
								nextPlayer = blue
							} else {
								nextPlayer = red
							}
							break
						}
					}
				}
			}
		}

		window.FillRect(0, 0, windowWidth, windowHeight, draw.White)
		for x := 0; x < boardWidth; x++ {
			for y := 0; y < boardHeight; y++ {
				color := draw.Black
				if winner == empty && x == mouseColumn && isFloor(x, y) {
					g := oscillateBetween(0.25, 0.75, time)
					if nextPlayer == red {
						color = draw.RGB(g, g/2, g/2)
					} else {
						color = draw.RGB(g/2, g/2, g)
					}
				}
				if board[x][y] == red {
					color = draw.Red
				}
				if board[x][y] == blue {
					color = draw.Blue
				}
				window.FillEllipse(
					margin+x*(tileSize+margin),
					margin+y*(tileSize+margin),
					tileSize,
					tileSize,
					color,
				)
			}
		}

		if winner != empty {
			var color draw.Color

			textAlpha := oscillateBetween(0, 1, time*0.7)
			backgroundAlpha := oscillateBetween(0.75, 1, time*0.7)

			var firstLine string
			if winner == red {
				firstLine = "    Red Player Wins!   "
				color = draw.RGBA(1, 0, 0, textAlpha)
			} else if winner == blue {
				firstLine = "   Blue Player Wins!   "
				color = draw.RGBA(0, 0, 1, textAlpha)
			} else if winner == both {
				firstLine = "    Nobody Wins :-(    "
				color = draw.RGBA(1, 0, 1, textAlpha)
			}
			secondLine := "Right-click to restart."
			text := "\n " + firstLine + " \n\n " + secondLine + " \n"

			textW, textH := window.GetScaledTextSize(text, textScale)
			textX := (windowWidth - textW) / 2
			textY := (windowHeight - textH) / 2
			window.FillRect(textX, textY, textW, textH, draw.RGBA(1, 1, 1, backgroundAlpha))
			window.DrawScaledText(text, textX, textY, textScale, color)
		}
	})
}

func oscillateBetween(a, b, t float64) float32 {
	return float32(a + (math.Sin(t*2*math.Pi)+1)*0.5*(b-a))
}
