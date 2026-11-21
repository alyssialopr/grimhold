package game

// Weapon est l'arme du joueur.
type Weapon struct {
	Name string `json:"name"`
	Dice string `json:"dice"`
}

// Character est la fiche du joueur.
type Character struct {
	Name        string `json:"name"`
	HP          int    `json:"hp"`
	MaxHP       int    `json:"max_hp"`
	Level       int    `json:"level"`
	XP          int    `json:"xp"`
	AttackBonus int    `json:"attack_bonus"`
	ArmorClass  int    `json:"armor_class"`
	Weapon      Weapon `json:"weapon"`
}

// NewCharacter crée un personnage de niveau 1.
func NewCharacter(name string) Character {
	return Character{
		Name:        name,
		HP:          24,
		MaxHP:       24,
		Level:       1,
		AttackBonus: 4,
		ArmorClass:  13,
		Weapon:      Weapon{Name: "Longsword", Dice: "1d8+2"},
	}
}

// NextLevelXP renvoie l'XP nécessaire pour passer au niveau suivant : level² × 50.
func (c Character) NextLevelXP() int {
	return c.Level * c.Level * 50
}

// GainXP ajoute de l'XP et renvoie le nombre de niveaux gagnés.
// Chaque niveau donne +6 PV max, +1 au toucher et soigne complètement.
func (c *Character) GainXP(xp int) int {
	c.XP += xp
	gained := 0
	for c.XP >= c.NextLevelXP() {
		c.Level++
		c.MaxHP += 6
		c.AttackBonus++
		c.HP = c.MaxHP
		gained++
	}
	return gained
}
