// Package repl gère la boucle de commandes du jeu.
package repl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alyssialopr/grimhold/internal/game"
	"github.com/alyssialopr/grimhold/internal/open5e"
)

// errQuit signale à la boucle qu'il faut s'arrêter.
var errQuit = errors.New("quit")

// cliCommand décrit une commande. L'aide est générée à partir de ces champs.
type cliCommand struct {
	name        string
	args        string // ex: "<region>", affiché dans l'aide
	description string
	callback    func(s *Session, args []string) error
}

// Session contient tout ce dont les commandes ont besoin.
type Session struct {
	game     *game.Game
	client   *open5e.Client
	out      io.Writer
	savePath string
	commands map[string]cliCommand

	// pagination du bestiaire
	nextPage, prevPage string
}

// New crée une session prête à lancer.
func New(g *game.Game, client *open5e.Client, out io.Writer, savePath string) *Session {
	s := &Session{game: g, client: client, out: out, savePath: savePath}
	s.commands = commands()
	return s
}

// Run lit les commandes ligne par ligne jusqu'à "quit" ou la fin de l'entrée.
func (s *Session) Run(in io.Reader) error {
	scanner := bufio.NewScanner(in)
	s.println("Welcome to Grimhold. Type 'help' to see the commands.")
	for {
		fmt.Fprint(s.out, paint(bold, "grimhold > "))
		if !scanner.Scan() {
			break
		}
		if err := s.Exec(scanner.Text()); err != nil {
			if errors.Is(err, errQuit) {
				return nil
			}
			s.println(paint(red, "Error: ") + err.Error())
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	s.println("")
	return nil
}

// Exec analyse et exécute une ligne. Une ligne vide ne fait rien.
func (s *Session) Exec(line string) error {
	words := strings.Fields(strings.ToLower(line))
	if len(words) == 0 {
		return nil
	}
	cmd, ok := s.commands[words[0]]
	if !ok {
		return fmt.Errorf("unknown command %q (type 'help')", words[0])
	}
	args := words[1:]
	if cmd.args == "" && len(args) > 0 {
		return fmt.Errorf("usage: %s (takes no arguments)", cmd.name)
	}
	return cmd.callback(s, args)
}

func (s *Session) println(a ...any) {
	fmt.Fprintln(s.out, a...)
}

func (s *Session) printf(format string, a ...any) {
	fmt.Fprintf(s.out, format, a...)
}

// usage renvoie une erreur si le nombre d'arguments n'est pas le bon.
func usage(args []string, n int, cmd string) error {
	if len(args) != n {
		return fmt.Errorf("usage: %s", cmd)
	}
	return nil
}
