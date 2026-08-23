package tui

import (
	"fmt"
)

func HideCursor() {
	fmt.Print("\033[?25l")
}

func ShowCursor() {
	fmt.Print("\033[?25h")
}

// Moves the terminal cursor up N lines and clears the view buffer.
func ClearLines(n int) {
	if n > 0 {
		fmt.Printf("\033[%dA\033[J", n)
	}
}
