// Package world charge la carte du jeu depuis world.json.
package world

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
)

// Region est une zone de la carte.
type Region struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Danger      int      `json:"danger"`
	Foes        []string `json:"foes"`
	Exits       []string `json:"exits"`
}

// World contient toutes les régions et le point de départ.
type World struct {
	Start   string   `json:"start"`
	Regions []Region `json:"regions"`
	byID    map[string]*Region
}

// Load lit et vérifie le fichier de la carte.
func Load(path string) (*World, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read world: %w", err)
	}
	return Parse(data)
}

// Parse décode la carte et vérifie qu'elle est cohérente.
func Parse(data []byte) (*World, error) {
	var w World
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("decode world: %w", err)
	}
	w.byID = make(map[string]*Region, len(w.Regions))
	for i := range w.Regions {
		w.byID[w.Regions[i].ID] = &w.Regions[i]
	}
	if _, ok := w.byID[w.Start]; !ok {
		return nil, fmt.Errorf("start region %q does not exist", w.Start)
	}
	for _, r := range w.Regions {
		if len(r.Foes) == 0 {
			return nil, fmt.Errorf("region %q has no foes", r.ID)
		}
		for _, e := range r.Exits {
			if _, ok := w.byID[e]; !ok {
				return nil, fmt.Errorf("region %q: unknown exit %q", r.ID, e)
			}
		}
	}
	return &w, nil
}

// Region renvoie une région par son id.
func (w *World) Region(id string) (*Region, bool) {
	r, ok := w.byID[id]
	return r, ok
}

// CanTravel vérifie qu'on peut aller de `from` à `to`.
func (w *World) CanTravel(from, to string) error {
	cur, ok := w.byID[from]
	if !ok {
		return fmt.Errorf("unknown current region %q", from)
	}
	if _, ok := w.byID[to]; !ok {
		return fmt.Errorf("there is no place called %q", to)
	}
	if !slices.Contains(cur.Exits, to) {
		return fmt.Errorf("no path leads from %s to %q", cur.Name, to)
	}
	return nil
}
