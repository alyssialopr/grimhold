package game

import (
	"errors"
	"fmt"

	"github.com/alyssialopr/grimhold/internal/open5e"
)

// Foe est un monstre en cours de combat.
type Foe struct {
	Slug       string
	Name       string
	HP, MaxHP  int
	ArmorClass int
	CR         float64
	Attack     open5e.Action
}

// NewFoe transforme un monstre de l'API en adversaire.
// On garde la première action qui inflige des dégâts (pas "Multiattack").
func NewFoe(m open5e.Monster) *Foe {
	f := &Foe{
		Slug:       m.Slug,
		Name:       m.Name,
		HP:         max(m.HitPoints, 1),
		MaxHP:      max(m.HitPoints, 1),
		ArmorClass: m.ArmorClass,
		CR:         m.CR,
		Attack:     open5e.Action{Name: "Claw", AttackBonus: 2, DamageDice: "1d4"},
	}
	for _, a := range m.Actions {
		if _, err := ParseDice(a.DamageDice); err == nil {
			f.Attack = a
			break
		}
	}
	return f
}

// XPReward donne l'XP d'une victoire selon le challenge rating.
func XPReward(cr float64) int {
	return 25 + int(cr*100)
}

// Explore cherche une rencontre dans la région actuelle.
// Une fois sur quatre il ne se passe rien. Sinon on tire `danger` monstres
// et on garde le pire (le plus haut CR).
func (g *Game) Explore(src MonsterSource) (*Foe, error) {
	if g.Foe != nil {
		return nil, ErrInCombat
	}
	if g.Rng.IntN(4) == 0 {
		return nil, nil
	}
	region := g.Region()
	rolls := max(region.Danger, 1)

	var worst *open5e.Monster
	for i := 0; i < rolls; i++ {
		slug := region.Foes[g.Rng.IntN(len(region.Foes))]
		m, err := src.GetMonster(slug)
		if err != nil {
			return nil, fmt.Errorf("explore: %w", err)
		}
		if worst == nil || m.CR > worst.CR {
			worst = &m
		}
	}
	g.Foe = NewFoe(*worst)
	return g.Foe, nil
}

// attack fait un jet d'attaque : d20 + bonus contre la CA, puis les dégâts si ça touche.
func (g *Game) attack(bonus, targetAC int, dice string) (hit bool, roll, dmg int, err error) {
	d, err := ParseDice(dice)
	if err != nil {
		return false, 0, 0, err
	}
	roll = d20(g.Rng)
	if roll+bonus < targetAC {
		return false, roll, 0, nil
	}
	return true, roll, max(d.Roll(g.Rng), 1), nil
}

// foeDice renvoie l'expression de dégâts complète de l'attaque du monstre.
func (f *Foe) foeDice() string {
	d, _ := ParseDice(f.Attack.DamageDice) // déjà validé dans NewFoe
	d.Bonus += f.Attack.DamageBonus
	return d.String()
}

// FightRound joue un tour de combat et renvoie le récit du tour.
func (g *Game) FightRound() ([]string, error) {
	if g.Foe == nil {
		return nil, errors.New("there is nothing to fight; try explore")
	}
	f := g.Foe
	c := &g.State.Character
	var log []string

	// Le joueur attaque.
	hit, roll, dmg, err := g.attack(c.AttackBonus, f.ArmorClass, c.Weapon.Dice)
	if err != nil {
		return nil, fmt.Errorf("player attack: %w", err)
	}
	if hit {
		f.HP -= dmg
		log = append(log, fmt.Sprintf("You roll %d+%d and hit the %s with your %s for %d damage.",
			roll, c.AttackBonus, f.Name, c.Weapon.Name, dmg))
	} else {
		log = append(log, fmt.Sprintf("You roll %d+%d and miss (AC %d).", roll, c.AttackBonus, f.ArmorClass))
	}
	if f.HP <= 0 {
		return append(log, g.victory()...), nil
	}

	// Le monstre riposte.
	log = append(log, g.foeStrikes(false)...)
	return log, nil
}

// Flee tente de fuir : d20+2 contre 12. En cas d'échec, le monstre frappe gratuitement.
func (g *Game) Flee() ([]string, error) {
	if g.Foe == nil {
		return nil, errors.New("there is nothing to flee from")
	}
	roll := d20(g.Rng)
	if roll+2 >= 12 {
		name := g.Foe.Name
		g.Foe = nil
		return []string{fmt.Sprintf("You roll %d+2 and slip away from the %s.", roll, name)}, nil
	}
	log := []string{fmt.Sprintf("You roll %d+2 and stumble while fleeing!", roll)}
	return append(log, g.foeStrikes(true)...), nil
}

// foeStrikes fait attaquer le monstre. free=true signifie que le coup touche forcément.
func (g *Game) foeStrikes(free bool) []string {
	f := g.Foe
	c := &g.State.Character
	ac := c.ArmorClass
	if free {
		ac = 0
	}
	hit, roll, dmg, err := g.attack(f.Attack.AttackBonus, ac, f.foeDice())
	if err != nil {
		// impossible en pratique : les dés ont été validés dans NewFoe
		return []string{fmt.Sprintf("The %s hesitates (%v).", f.Name, err)}
	}
	if !hit {
		return []string{fmt.Sprintf("The %s attacks with %s (%d+%d) and misses.",
			f.Name, f.Attack.Name, roll, f.Attack.AttackBonus)}
	}
	c.HP -= dmg
	log := []string{fmt.Sprintf("The %s hits you with %s for %d damage.", f.Name, f.Attack.Name, dmg)}
	if c.HP <= 0 {
		log = append(log, g.defeat()...)
	}
	return log
}

func (g *Game) victory() []string {
	f := g.Foe
	g.Foe = nil
	xp := XPReward(f.CR)
	gold := g.Rng.IntN(int(f.CR*10)+5) + 1
	g.State.Gold += gold
	g.State.Slain[f.Name]++

	log := []string{fmt.Sprintf("The %s falls. You gain %d XP and %d gold.", f.Name, xp, gold)}
	c := &g.State.Character
	if n := c.GainXP(xp); n > 0 {
		log = append(log, fmt.Sprintf("Level up! You are now level %d (max HP %d, attack +%d).",
			c.Level, c.MaxHP, c.AttackBonus))
	}
	return log
}

// defeat ramène le joueur au départ avec 1 PV. La sauvegarde sur disque n'est pas touchée.
func (g *Game) defeat() []string {
	name := g.Foe.Name
	g.Foe = nil
	g.State.Character.HP = 1
	g.State.Location = g.World.Start
	g.State.RestedHere = false
	start := g.Region()
	return []string{
		fmt.Sprintf("The %s strikes you down. Everything goes dark...", name),
		fmt.Sprintf("You wake up at %s with 1 HP and your pride wounded.", start.Name),
	}
}
