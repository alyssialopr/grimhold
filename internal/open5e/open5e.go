// Package open5e est un petit client pour l'API https://api.open5e.com.
package open5e

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/alyssialopr/grimhold/internal/cache"
)

// BaseURL est l'adresse de l'API v1.
const BaseURL = "https://api.open5e.com/v1"

// ErrNotFound est renvoyée quand le monstre n'existe pas.
var ErrNotFound = errors.New("monster not found")

// Action est une attaque ou capacité d'un monstre.
type Action struct {
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	AttackBonus int    `json:"attack_bonus"`
	DamageDice  string `json:"damage_dice"`
	DamageBonus int    `json:"damage_bonus"`
}

// Monster contient les champs de l'API qui nous intéressent.
type Monster struct {
	Slug            string   `json:"slug"`
	Name            string   `json:"name"`
	Size            string   `json:"size"`
	Type            string   `json:"type"`
	Alignment       string   `json:"alignment"`
	ArmorClass      int      `json:"armor_class"`
	HitPoints       int      `json:"hit_points"`
	HitDice         string   `json:"hit_dice"`
	ChallengeRating string   `json:"challenge_rating"`
	CR              float64  `json:"cr"`
	Strength        int      `json:"strength"`
	Dexterity       int      `json:"dexterity"`
	Constitution    int      `json:"constitution"`
	Intelligence    int      `json:"intelligence"`
	Wisdom          int      `json:"wisdom"`
	Charisma        int      `json:"charisma"`
	Actions         []Action `json:"actions"`
}

// MonsterPage est une page de résultats paginée.
// "next" et "previous" valent null quand il n'y a pas de page : ils sont alors vides.
type MonsterPage struct {
	Count    int       `json:"count"`
	Next     string    `json:"next"`
	Previous string    `json:"previous"`
	Results  []Monster `json:"results"`
}

// Client interroge l'API en passant par un cache.
type Client struct {
	baseURL string
	http    *http.Client
	cache   *cache.Cache
}

// NewClient crée un client. baseURL peut pointer vers un serveur de test.
func NewClient(baseURL string, c *cache.Cache) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
		cache:   c,
	}
}

// get renvoie le corps de la réponse, depuis le cache si possible.
func (c *Client) get(u string) ([]byte, error) {
	if body, ok := c.cache.Get(u); ok {
		return body, nil
	}
	resp, err := c.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request %s: unexpected status %s", u, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	c.cache.Add(u, body)
	return body, nil
}

// ListMonsters renvoie une page du bestiaire. Si pageURL est vide,
// on charge la première page (monstres du SRD uniquement).
func (c *Client) ListMonsters(pageURL string) (MonsterPage, error) {
	if pageURL == "" {
		pageURL = c.baseURL + "/monsters/?document__slug=wotc-srd&limit=10"
	}
	var page MonsterPage
	body, err := c.get(pageURL)
	if err != nil {
		return page, fmt.Errorf("list monsters: %w", err)
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return page, fmt.Errorf("decode monster page: %w", err)
	}
	return page, nil
}

// GetMonster récupère un monstre par son slug (ex: "goblin").
func (c *Client) GetMonster(slug string) (Monster, error) {
	var m Monster
	body, err := c.get(c.baseURL + "/monsters/" + url.PathEscape(slug) + "/")
	if err != nil {
		return m, fmt.Errorf("get monster %q: %w", slug, err)
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return m, fmt.Errorf("decode monster %q: %w", slug, err)
	}
	return m, nil
}

// Prefetch charge plusieurs monstres en parallèle pour remplir le cache.
func (c *Client) Prefetch(slugs []string) error {
	var wg sync.WaitGroup
	errs := make([]error, len(slugs))
	for i, slug := range slugs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = c.GetMonster(slug)
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}
