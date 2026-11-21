package repl

import (
	"fmt"
	"os"
	"strings"
)

// Codes ANSI écrits à la main. On les désactive si NO_COLOR est défini.
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	gray   = "\033[90m"
)

var useColor = os.Getenv("NO_COLOR") == ""

func paint(color, s string) string {
	if !useColor {
		return s
	}
	return color + s + reset
}

// hpBar dessine une barre de vie colorée : [██████····] 12/20
func hpBar(hp, maxHP int) string {
	const width = 20
	hp = max(hp, 0)
	filled := 0
	if maxHP > 0 {
		filled = min(hp*width/maxHP, width)
	}
	color := green
	switch {
	case hp*3 <= maxHP:
		color = red
	case hp*3 <= maxHP*2:
		color = yellow
	}
	bar := paint(color, strings.Repeat("█", filled)) + paint(gray, strings.Repeat("·", width-filled))
	return fmt.Sprintf("[%s] %d/%d", bar, hp, maxHP)
}

// dangerLabel affiche le niveau de danger avec une couleur.
func dangerLabel(d int) string {
	switch {
	case d <= 1:
		return paint(green, "low")
	case d == 2:
		return paint(yellow, "high")
	default:
		return paint(red, "deadly")
	}
}
