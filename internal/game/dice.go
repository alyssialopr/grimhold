package game

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

// Dice représente une expression comme "2d6+3".
type Dice struct {
	Count, Sides, Bonus int
}

// ParseDice lit une expression de dés : "1d8", "2d6+3", "1d4 - 1".
func ParseDice(s string) (Dice, error) {
	expr := strings.ReplaceAll(strings.ToLower(s), " ", "")
	var d Dice

	// On sépare le bonus éventuel (+N ou -N).
	sign := 1
	if i := strings.IndexAny(expr, "+-"); i >= 0 {
		if expr[i] == '-' {
			sign = -1
		}
		b, err := strconv.Atoi(expr[i+1:])
		if err != nil {
			return d, fmt.Errorf("bad dice bonus in %q: %w", s, err)
		}
		d.Bonus = sign * b
		expr = expr[:i]
	}

	count, sides, ok := strings.Cut(expr, "d")
	if !ok {
		return d, fmt.Errorf("bad dice expression %q", s)
	}
	var err error
	if d.Count, err = strconv.Atoi(count); err != nil || d.Count < 1 {
		return d, fmt.Errorf("bad dice count in %q", s)
	}
	if d.Sides, err = strconv.Atoi(sides); err != nil || d.Sides < 1 {
		return d, fmt.Errorf("bad dice sides in %q", s)
	}
	return d, nil
}

// Roll lance les dés avec le générateur donné.
func (d Dice) Roll(r *rand.Rand) int {
	total := d.Bonus
	for i := 0; i < d.Count; i++ {
		total += r.IntN(d.Sides) + 1
	}
	return total
}

func (d Dice) String() string {
	switch {
	case d.Bonus > 0:
		return fmt.Sprintf("%dd%d+%d", d.Count, d.Sides, d.Bonus)
	case d.Bonus < 0:
		return fmt.Sprintf("%dd%d%d", d.Count, d.Sides, d.Bonus)
	}
	return fmt.Sprintf("%dd%d", d.Count, d.Sides)
}

// d20 lance un dé à 20 faces.
func d20(r *rand.Rand) int {
	return r.IntN(20) + 1
}
