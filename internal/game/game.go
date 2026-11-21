// Package game contient les règles : déplacement, rencontres, combat, sauvegarde.
package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/alyssialopr/grimhold/internal/open5e"
	"github.com/alyssialopr/grimhold/internal/world"
)

// ErrNoSave est renvoyée par Load quand le fichier de sauvegarde n'existe pas.
var ErrNoSave = errors.New("no save found")

// ErrInCombat empêche certaines actions pendant un combat.
var ErrInCombat = errors.New("you are in combat: fight or flee first")

// MonsterSource permet de récupérer un monstre (le client Open5e, ou un faux en test).
type MonsterSource interface {
	GetMonster(slug string) (open5e.Monster, error)
}

// State est tout ce qui est sauvegardé dans save.json.
type State struct {
	Location   string         `json:"location"`
	Character  Character      `json:"character"`
	Slain      map[string]int `json:"slain"`
	Gold       int            `json:"gold"`
	RestedHere bool           `json:"rested_here"`
}

// Game regroupe l'état, la carte, le générateur aléatoire et le combat en cours.
type Game struct {
	State State
	World *world.World
	Rng   *rand.Rand
	Foe   *Foe // combat en cours (jamais sauvegardé)
}

// New démarre une nouvelle partie au point de départ de la carte.
func New(w *world.World, rng *rand.Rand) *Game {
	return &Game{
		World: w,
		Rng:   rng,
		State: State{
			Location:  w.Start,
			Character: NewCharacter("Wanderer"),
			Slain:     make(map[string]int),
		},
	}
}

// Region renvoie la région actuelle.
func (g *Game) Region() *world.Region {
	r, _ := g.World.Region(g.State.Location)
	return r
}

// Travel déplace le joueur vers une région voisine.
func (g *Game) Travel(to string) (*world.Region, error) {
	if g.Foe != nil {
		return nil, ErrInCombat
	}
	if err := g.World.CanTravel(g.State.Location, to); err != nil {
		return nil, fmt.Errorf("cannot travel: %w", err)
	}
	g.State.Location = to
	g.State.RestedHere = false
	return g.Region(), nil
}

// Rest soigne complètement, une seule fois par visite de région.
func (g *Game) Rest() error {
	if g.Foe != nil {
		return ErrInCombat
	}
	if g.State.RestedHere {
		return errors.New("you already rested here; travel somewhere else first")
	}
	g.State.Character.HP = g.State.Character.MaxHP
	g.State.RestedHere = true
	return nil
}

// Save écrit l'état dans un fichier temporaire puis le renomme :
// si le programme plante pendant l'écriture, l'ancienne sauvegarde reste intacte.
func (g *Game) Save(path string) error {
	if g.Foe != nil {
		return ErrInCombat
	}
	data, err := json.MarshalIndent(g.State, "", "  ")
	if err != nil {
		return fmt.Errorf("encode save: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "save-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp save: %w", err)
	}
	defer os.Remove(tmp.Name()) // sans effet si le renommage a réussi
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write save: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close save: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replace save: %w", err)
	}
	return nil
}

// Load restaure l'état depuis un fichier de sauvegarde.
func (g *Game) Load(path string) error {
	if g.Foe != nil {
		return ErrInCombat
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNoSave
	}
	if err != nil {
		return fmt.Errorf("read save: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("decode save: %w", err)
	}
	if _, ok := g.World.Region(s.Location); !ok {
		return fmt.Errorf("save points to unknown region %q", s.Location)
	}
	if s.Slain == nil {
		s.Slain = make(map[string]int)
	}
	g.State = s
	return nil
}
