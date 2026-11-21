package repl

import (
	"errors"
	"sort"
	"strings"

	"github.com/alyssialopr/grimhold/internal/game"
)

// commands enregistre toutes les commandes du jeu.
func commands() map[string]cliCommand {
	list := []cliCommand{
		{"help", "", "Show this message", cmdHelp},
		{"map", "", "List the known regions", cmdMap},
		{"travel", "<region>", "Travel to a connected region", cmdTravel},
		{"look", "", "Describe your surroundings", cmdLook},
		{"explore", "", "Wander the region (risk an encounter)", cmdExplore},
		{"fight", "", "Play one round of combat", cmdFight},
		{"flee", "", "Try to run away (failure means a free hit)", cmdFlee},
		{"rest", "", "Heal fully (once per region visit)", cmdRest},
		{"character", "", "Show your character sheet", cmdCharacter},
		{"slain", "", "Show the monsters you have killed", cmdSlain},
		{"bestiary", "", "List monsters from Open5e (first page)", cmdBestiary},
		{"next", "", "Next bestiary page", cmdNext},
		{"prev", "", "Previous bestiary page", cmdPrev},
		{"inspect", "<monster>", "Show a monster's stat block", cmdInspect},
		{"save", "", "Save the game to save.json", cmdSave},
		{"load", "", "Load the game from save.json", cmdLoad},
		{"quit", "", "Leave Grimhold", cmdQuit},
	}
	m := make(map[string]cliCommand, len(list))
	for _, c := range list {
		m[c.name] = c
	}
	return m
}

func cmdHelp(s *Session, args []string) error {
	s.println("Grimhold — a dungeon crawler")
	s.println("")
	s.println("Commands:")
	names := make([]string, 0, len(s.commands))
	for name := range s.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := s.commands[name]
		s.printf("  %-24s %s\n", strings.TrimSpace(c.name+" "+c.args), c.description)
	}
	return nil
}

func cmdMap(s *Session, args []string) error {
	for _, r := range s.game.World.Regions {
		marker := "  "
		if r.ID == s.game.State.Location {
			marker = paint(bold, "> ")
		}
		s.printf("%s%-18s danger %-6s exits: %s\n", marker, r.ID, dangerLabel(r.Danger), strings.Join(r.Exits, ", "))
	}
	return nil
}

func cmdTravel(s *Session, args []string) error {
	if err := usage(args, 1, "travel <region>"); err != nil {
		return err
	}
	r, err := s.game.Travel(args[0])
	if err != nil {
		return err
	}
	s.printf("You travel to %s. %s\n", paint(bold, r.Name), r.Description)

	// Bonus : on précharge les monstres de la région en parallèle.
	if err := s.client.Prefetch(r.Foes); err != nil {
		s.println(paint(yellow, "Warning: could not prefetch some foes: ") + err.Error())
	}
	return nil
}

func cmdLook(s *Session, args []string) error {
	r := s.game.Region()
	s.println(paint(bold, r.Name))
	s.println(r.Description)
	s.println("Exits:", strings.Join(r.Exits, ", "))
	s.println("Danger:", dangerLabel(r.Danger))
	if f := s.game.Foe; f != nil {
		s.printf("A %s stands before you. %s\n", f.Name, hpBar(f.HP, f.MaxHP))
	}
	return nil
}

func cmdExplore(s *Session, args []string) error {
	foe, err := s.game.Explore(s.client)
	if err != nil {
		return err
	}
	if foe == nil {
		s.println("You wander around. Nothing but silence... for now.")
		return nil
	}
	s.printf("A %s emerges! (AC %d, CR %g) %s\n", paint(red, foe.Name), foe.ArmorClass, foe.CR, hpBar(foe.HP, foe.MaxHP))
	s.println("Type 'fight' or 'flee'.")
	return nil
}

func cmdFight(s *Session, args []string) error {
	foe := s.game.Foe
	log, err := s.game.FightRound()
	if err != nil {
		return err
	}
	s.printLog(log)
	if s.game.Foe != nil {
		s.printf("  %s %s   You %s\n", foe.Name, hpBar(foe.HP, foe.MaxHP),
			hpBar(s.game.State.Character.HP, s.game.State.Character.MaxHP))
	}
	return nil
}

func cmdFlee(s *Session, args []string) error {
	log, err := s.game.Flee()
	if err != nil {
		return err
	}
	s.printLog(log)
	return nil
}

func (s *Session) printLog(log []string) {
	for _, l := range log {
		s.println("  " + l)
	}
}

func cmdRest(s *Session, args []string) error {
	if err := s.game.Rest(); err != nil {
		return err
	}
	c := s.game.State.Character
	s.println("You rest by a small fire.", hpBar(c.HP, c.MaxHP))
	return nil
}

func cmdCharacter(s *Session, args []string) error {
	st := s.game.State
	c := st.Character
	s.println(paint(bold, c.Name))
	s.printf("  Level   %d (XP %d/%d)\n", c.Level, c.XP, c.NextLevelXP())
	s.printf("  HP      %s\n", hpBar(c.HP, c.MaxHP))
	s.printf("  Attack  +%d   AC %d\n", c.AttackBonus, c.ArmorClass)
	s.printf("  Weapon  %s (%s)\n", c.Weapon.Name, c.Weapon.Dice)
	s.printf("  Gold    %d\n", st.Gold)
	return nil
}

func cmdSlain(s *Session, args []string) error {
	slain := s.game.State.Slain
	if len(slain) == 0 {
		s.println("You have not slain anything yet.")
		return nil
	}
	names := make([]string, 0, len(slain))
	for n := range slain {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		s.printf("  %-20s x%d\n", n, slain[n])
	}
	return nil
}

func cmdBestiary(s *Session, args []string) error {
	return s.showPage("")
}

func cmdNext(s *Session, args []string) error {
	if s.nextPage == "" {
		return errors.New("no next page (use 'bestiary' first)")
	}
	return s.showPage(s.nextPage)
}

func cmdPrev(s *Session, args []string) error {
	if s.prevPage == "" {
		return errors.New("no previous page")
	}
	return s.showPage(s.prevPage)
}

func (s *Session) showPage(url string) error {
	page, err := s.client.ListMonsters(url)
	if err != nil {
		return err
	}
	for _, m := range page.Results {
		s.printf("  %-28s CR %-5s HP %-4d AC %d\n", m.Slug, m.ChallengeRating, m.HitPoints, m.ArmorClass)
	}
	s.nextPage, s.prevPage = page.Next, page.Previous
	s.printf("(%d monsters in total — 'next' / 'prev' to browse)\n", page.Count)
	return nil
}

func cmdInspect(s *Session, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: inspect <monster>")
	}
	slug := strings.Join(args, "-") // "vampire spawn" -> "vampire-spawn"
	m, err := s.client.GetMonster(slug)
	if err != nil {
		return err
	}
	s.println(paint(bold, m.Name))
	s.printf("  %s %s, %s\n", m.Size, m.Type, m.Alignment)
	s.printf("  AC %d   HP %d (%s)   CR %s\n", m.ArmorClass, m.HitPoints, m.HitDice, m.ChallengeRating)
	s.printf("  STR %d  DEX %d  CON %d  INT %d  WIS %d  CHA %d\n",
		m.Strength, m.Dexterity, m.Constitution, m.Intelligence, m.Wisdom, m.Charisma)
	if len(m.Actions) > 0 {
		s.println("  Actions:")
		for _, a := range m.Actions {
			s.printf("    - %s: %s\n", paint(bold, a.Name), a.Desc)
		}
	}
	return nil
}

func cmdSave(s *Session, args []string) error {
	if err := s.game.Save(s.savePath); err != nil {
		return err
	}
	s.println("Game saved to", s.savePath)
	return nil
}

func cmdLoad(s *Session, args []string) error {
	err := s.game.Load(s.savePath)
	if errors.Is(err, game.ErrNoSave) {
		s.println("No save found. Use 'save' first.")
		return nil
	}
	if err != nil {
		return err
	}
	s.printf("Game loaded. You are at %s.\n", s.game.Region().Name)
	return nil
}

func cmdQuit(s *Session, args []string) error {
	s.println("The fog closes behind you. Farewell.")
	return errQuit
}
