package game

import (
	"errors"
	"math/rand/v2"
	"path/filepath"
	"testing"

	"github.com/alyssialopr/grimhold/internal/open5e"
	"github.com/alyssialopr/grimhold/internal/world"
)

const testWorld = `{"start":"camp","regions":[
 {"id":"camp","name":"Camp","danger":1,"foes":["goblin"],"exits":["cave"]},
 {"id":"cave","name":"Cave","danger":2,"foes":["goblin","ogre"],"exits":["camp"]}]}`

// fakeSource remplace l'API Open5e dans les tests.
type fakeSource map[string]open5e.Monster

func (f fakeSource) GetMonster(slug string) (open5e.Monster, error) {
	m, ok := f[slug]
	if !ok {
		return m, open5e.ErrNotFound
	}
	return m, nil
}

var monsters = fakeSource{
	"goblin": {Slug: "goblin", Name: "Goblin", HitPoints: 7, ArmorClass: 15, CR: 0.25,
		Actions: []open5e.Action{{Name: "Scimitar", AttackBonus: 4, DamageDice: "1d6", DamageBonus: 2}}},
	"ogre": {Slug: "ogre", Name: "Ogre", HitPoints: 59, ArmorClass: 11, CR: 2,
		Actions: []open5e.Action{{Name: "Greatclub", AttackBonus: 6, DamageDice: "2d8", DamageBonus: 4}}},
}

func newTestGame(t *testing.T, seed uint64) *Game {
	t.Helper()
	w, err := world.Parse([]byte(testWorld))
	if err != nil {
		t.Fatal(err)
	}
	return New(w, rand.New(rand.NewPCG(seed, seed)))
}

func TestParseDice(t *testing.T) {
	tests := []struct {
		in      string
		want    Dice
		wantErr bool
	}{
		{"1d8", Dice{1, 8, 0}, false},
		{"2d6+3", Dice{2, 6, 3}, false},
		{"1d4 - 1", Dice{1, 4, -1}, false},
		{"18d10+36", Dice{18, 10, 36}, false},
		{"d6", Dice{}, true},
		{"banana", Dice{}, true},
		{"", Dice{}, true},
	}
	for _, tt := range tests {
		got, err := ParseDice(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseDice(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseDice(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

// Avec un seed fixe, le combat est reproductible ; un joueur très fort
// gagne toujours et la victoire est bien enregistrée.
func TestFightVictory(t *testing.T) {
	g := newTestGame(t, 42)
	g.State.Character.AttackBonus = 100 // touche à coup sûr
	g.State.Character.Weapon.Dice = "1d1+50"
	g.Foe = NewFoe(monsters["goblin"])

	if _, err := g.FightRound(); err != nil {
		t.Fatal(err)
	}
	if g.Foe != nil {
		t.Fatal("goblin should be dead")
	}
	if g.State.Slain["Goblin"] != 1 {
		t.Fatalf("slain = %v, want Goblin:1", g.State.Slain)
	}
	if g.State.Character.XP != XPReward(0.25) {
		t.Fatalf("XP = %d, want %d", g.State.Character.XP, XPReward(0.25))
	}
	if g.State.Gold == 0 {
		t.Fatal("expected some gold")
	}
}

// Une défaite renvoie au départ avec 1 PV, sans toucher à la sauvegarde.
func TestDefeatSendsPlayerHome(t *testing.T) {
	g := newTestGame(t, 1)
	save := filepath.Join(t.TempDir(), "save.json")
	if err := g.Save(save); err != nil {
		t.Fatal(err)
	}

	if _, err := g.Travel("cave"); err != nil {
		t.Fatal(err)
	}
	g.State.Character.HP = 1
	g.State.Character.ArmorClass = -100 // le monstre touche à coup sûr
	g.State.Character.AttackBonus = -100
	g.Foe = NewFoe(monsters["ogre"])

	if _, err := g.FightRound(); err != nil {
		t.Fatal(err)
	}
	if g.Foe != nil || g.State.Location != "camp" || g.State.Character.HP != 1 {
		t.Fatalf("after defeat: foe=%v loc=%s hp=%d", g.Foe, g.State.Location, g.State.Character.HP)
	}

	// La sauvegarde est toujours chargeable et intacte.
	g2 := newTestGame(t, 1)
	if err := g2.Load(save); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if g2.State.Character.HP != 24 {
		t.Fatalf("save corrupted: hp=%d", g2.State.Character.HP)
	}
}

func TestExploreKeepsWorstFoe(t *testing.T) {
	// On essaie plusieurs seeds : en zone de danger 2, si un ogre est tiré
	// il doit être gardé plutôt que le gobelin.
	sawOgre := false
	for seed := uint64(0); seed < 50; seed++ {
		g := newTestGame(t, seed)
		g.State.Location = "cave"
		foe, err := g.Explore(monsters)
		if err != nil {
			t.Fatal(err)
		}
		if foe != nil && foe.Name == "Ogre" {
			sawOgre = true
		}
	}
	if !sawOgre {
		t.Fatal("never met the ogre in 50 explores")
	}
}

func TestRestOncePerVisit(t *testing.T) {
	g := newTestGame(t, 1)
	if err := g.Rest(); err != nil {
		t.Fatal(err)
	}
	if err := g.Rest(); err == nil {
		t.Fatal("second rest in the same region should fail")
	}
	if _, err := g.Travel("cave"); err != nil {
		t.Fatal(err)
	}
	if err := g.Rest(); err != nil {
		t.Fatalf("rest after travel: %v", err)
	}
}

func TestLoadWithoutSave(t *testing.T) {
	g := newTestGame(t, 1)
	err := g.Load(filepath.Join(t.TempDir(), "missing.json"))
	if !errors.Is(err, ErrNoSave) {
		t.Fatalf("got %v, want ErrNoSave", err)
	}
}

func TestLevelUp(t *testing.T) {
	c := NewCharacter("test")
	if n := c.GainXP(50); n != 1 || c.Level != 2 || c.MaxHP != 30 || c.AttackBonus != 5 {
		t.Fatalf("after 50 XP: gained=%d %+v", n, c)
	}
}
