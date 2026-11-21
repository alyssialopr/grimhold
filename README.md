# Grimhold

Un petit dungeon crawler en mode texte, écrit en Go. On se déplace entre des
régions maudites, on tombe sur des monstres, on les combat aux dés et on tient
un bestiaire de ses victimes. Les monstres viennent de l'API
[Open5e](https://open5e.com) (pas de clé nécessaire), la carte vient de `world.json`.

Uniquement la bibliothèque standard, Go 1.22+.

## Compiler et lancer

```bash
go build -o grimhold ./cmd/grimhold   # grimhold.exe sous Windows
./grimhold                            # nouvelle partie
./grimhold --continue                 # reprend save.json
```

Options : `--world <fichier>` (défaut `world.json`), `--save <fichier>` (défaut `save.json`).
Pour désactiver les couleurs : `NO_COLOR=1 ./grimhold`.

Tests et vérifications :

```bash
go test ./...
go vet ./...
gofmt -l .
```

## Commandes

Tapez `help` en jeu : la liste est générée automatiquement à partir de la map de commandes.

| Commande | Rôle |
|---|---|
| `map` / `look` | lister les régions / décrire la région actuelle |
| `travel <region>` | aller dans une région voisine (seulement par une sortie) |
| `explore` | chercher une rencontre |
| `fight` / `flee` | jouer un tour de combat / tenter de fuir |
| `rest` | soin complet, une fois par visite de région |
| `character` / `slain` | fiche du personnage / monstres tués |
| `bestiary`, `next`, `prev` | parcourir les monstres d'Open5e page par page |
| `inspect <monster>` | fiche d'un monstre (ex. `inspect vampire spawn`) |
| `save` / `load` | sauvegarder / charger `save.json` |
| `quit` | quitter |

## Organisation du code

```
cmd/grimhold/main.go   point d'entrée (flags, assemblage des packages)
internal/world         chargement et vérification de world.json, règles de déplacement
internal/cache         cache mémoire avec TTL, mutex et goroutine de nettoyage
internal/open5e        client HTTP de l'API (structs typées, pagination, cache)
internal/game          personnage, dés, rencontres, combat, sauvegarde
internal/repl          boucle de commandes et affichage (couleurs ANSI)
```

## Comment ça marche

**REPL.** Chaque commande est un `cliCommand{name, args, description, callback}`
rangé dans une `map[string]cliCommand`. La ligne tapée est découpée en mots, le
premier mot sert à trouver la commande. Commande inconnue, mauvais nombre
d'arguments ou ligne vide : on affiche un message et la boucle continue. Les
erreurs des commandes sont simplement affichées, le programme ne plante jamais.

**Cache.** Les réponses de l'API sont gardées 5 minutes dans une `map` protégée
par un `sync.Mutex`. Une goroutine (`reapLoop`) se réveille toutes les minutes
avec un `time.Ticker` et supprime les entrées expirées. L'horloge est
injectable pour pouvoir tester l'expiration sans attendre.

**Rencontres.** `explore` a une chance sur quatre de ne rien trouver. Sinon on
tire un monstre au hasard dans la liste `foes` de la région autant de fois que
le niveau de `danger`, et on garde le plus dangereux (plus haut CR).

**Combat.** Un `fight` = un tour :
- le joueur lance `d20 + bonus d'attaque` contre la CA du monstre, et s'il touche il lance les dés de son arme ;
- le monstre riposte de la même façon avec sa première attaque qui a des dés de dégâts ;
- `flee` : `d20 + 2` contre 12, en cas d'échec le monstre frappe gratuitement (touche automatique).

**Progression.** Une victoire rapporte `25 + CR × 100` XP, un peu d'or et une
entrée dans le bestiaire des monstres tués. Il faut `niveau² × 50` XP pour monter
de niveau ; chaque niveau donne +6 PV max, +1 en attaque et un soin complet.

**Mort.** On se réveille à la croisée des chemins avec 1 PV. La mort ne touche
qu'à l'état en mémoire : rien n'est écrit sur le disque.

**Sauvegarde.** `save` écrit l'état (lieu, personnage, bestiaire, or, repos)
dans un fichier temporaire puis le renomme en `save.json`, donc une sauvegarde
n'est jamais à moitié écrite. On ne peut pas sauvegarder en plein combat. Si
aucune sauvegarde n'existe, `load` le dit simplement.

## Bonus réalisés

- **Suite de tests (+6)** : `go test ./...`, aucun test n'utilise le réseau.
  - `cache` : expiration avec une fausse horloge, et test du reaper ;
  - `open5e` : décodage, pagination, 404 et cache vérifiés contre un `httptest.Server` ;
  - `game` : parsing des dés (table-driven), combat avec un `rand` à seed fixe, défaite sans corrompre la sauvegarde, repos, montée de niveau ;
  - `world` : déplacements autorisés / refusés (table-driven).
- **Faster dungeon (+4)** : à chaque `travel`, les monstres de la nouvelle région
  sont préchargés en parallèle (`Client.Prefetch`, une goroutine par monstre +
  `sync.WaitGroup`, erreurs regroupées avec `errors.Join`). Le `explore` suivant
  est alors instantané grâce au cache.
- **Couleurs ANSI (+2)** : codes écrits à la main dans `internal/repl/color.go`,
  barres de PV vertes / jaunes / rouges et niveau de danger coloré.

## Exemple de partie

```text
Welcome to Grimhold. Type 'help' to see the commands.
grimhold > travel barrow-hills
Error: cannot travel: no path leads from The Crossroads to "barrow-hills"
grimhold > look
The Crossroads
A leaning signpost, a dead fire, and crows that watch a little too closely.
Exits: ashen-marsh, whispering-woods, old-mine
Danger: low
grimhold > travel ashen-marsh
You travel to Ashen Marsh. Fog, iron, and things that breathe under the water.
grimhold > explore
A Ghoul emerges! (AC 12, CR 1) [████████████████████] 22/22
Type 'fight' or 'flee'.
grimhold > travel crossroads
Error: you are in combat: fight or flee first
grimhold > fight
  You roll 13+4 and hit the Ghoul with your Longsword for 10 damage.
  The Ghoul attacks with Bite (6+2) and misses.
  Ghoul [██████████··········] 12/22   You [████████████████████] 24/24
grimhold > fight
  You roll 14+4 and hit the Ghoul with your Longsword for 9 damage.
  The Ghoul hits you with Bite for 9 damage.
  Ghoul [██··················] 3/22   You [████████████········] 15/24
grimhold > fight
  You roll 12+4 and hit the Ghoul with your Longsword for 6 damage.
  The Ghoul falls. You gain 125 XP and 15 gold.
  Level up! You are now level 2 (max HP 30, attack +5).
grimhold > slain
  Ghoul                x1
grimhold > save
Game saved to save.json
grimhold > quit
The fog closes behind you. Farewell.
```
