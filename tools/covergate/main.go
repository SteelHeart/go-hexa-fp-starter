// Programme covergate applique les cliquets de couverture, à l'identique en
// local et en CI.
//
// # Pourquoi un programme Go et pas trois lignes de shell
//
// Les seuils vivaient à TROIS endroits, avec TROIS valeurs : `COVER_THRESHOLD:
// 80` dans le Taskfile (jamais lu par aucune tâche), `>= 70` en CI, `>= 90` pour
// le cœur. Trois sources de vérité qui se contredisent valent moins qu'une seule
// qui refuse.
//
// Un programme Go, appelé par la CI ET par `task`, ne peut pas dériver : il n'y a
// plus qu'un seul endroit où le seuil est écrit, et il est compilé.
//
// # Le défaut qu'il corrige
//
// Le cliquet global exigeait 70 % d'un profil produit par `go test ./...` SANS
// TAG. Or ce lot ne peut, par construction, exécuter aucune ligne d'un pilote
// Postgres ou Redis : `toolchain.md` garantit qu'un test sans tag n'exige aucun
// service. Le seuil était donc inatteignable — non par manque de rigueur, mais
// parce qu'il mesurait du code que ce lot ne peut pas atteindre.
//
// Un seuil inatteignable ne protège rien : il devient rouge en permanence, on
// l'ignore, puis on l'abaisse « pour débloquer la CI ». C'est le scénario que
// rules/tests.md §5 interdit explicitement.
//
// La réponse n'est PAS d'abaisser la barre. C'est de la mesurer là où elle
// s'applique, et de dire ce qui en sort. Trois cliquets :
//
//  1. périmètre UNITAIRE  ≥ 70 %   — le code que ce lot peut atteindre
//  2. cœur métier         ≥ 90 %   — domain/ et application/, pondéré
//  3. code PRODUIT        cliquet  — ne redescend jamais
//
// Le troisième est ce qui empêche l'exclusion d'être une dissimulation : le
// chiffre incluant TOUT est encore gardé, il ne peut que monter. Ajouter du code
// non couvert reste bloqué.
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Les trois seuils. UN seul endroit dans tout le dépôt.
const (
	// unitThreshold s'applique au code que `go test ./...` sans tag peut atteindre.
	unitThreshold = 70.0

	// coreThreshold s'applique à domain/ et application/ — les règles métier.
	// Plancher, pas objectif : la couverture mesure ce qui est EXÉCUTÉ, pas ce
	// qui est VÉRIFIÉ.
	coreThreshold = 90.0

	// productFloor est un CLIQUET, pas un objectif. Il porte sur TOUT le code
	// produit — pilotes Postgres et Redis compris, `cmd/` compris — et il ne
	// descend jamais. Le relever fait partie de toute PR qui couvre du code
	// jusque-là non couvert.
	//
	// C'est ce cliquet qui empêche le périmètre unitaire réduit d'être une
	// dissimulation : le chiffre incluant tout reste gardé. Ajouter du code non
	// couvert le fait baisser, donc échouer.
	//
	// Mesuré à 56,9 % le 2026-07-26. Posé à 56 pour absorber l'écart de mesure
	// entre `-covermode=count` (local) et `atomic` (CI, avec -race).
	productFloor = 56.0
)

// buildTooling est le SEUL chemin hors du cliquet de code produit.
//
// Il n'y est pas parce qu'il serait moins important, mais parce qu'un cliquet
// qui inclurait l'outillage de build punirait le fait d'en écrire : ajouter cet
// outil-ci a fait baisser le total global de 56,9 % à 54,0 % sans qu'une seule
// ligne de code produit change de couverture.
const buildTooling = "tools/"

// exclusion nomme un chemin hors du périmètre UNITAIRE, et POURQUOI.
//
// Aucune entrée sans raison écrite : c'est ce qui distingue un périmètre assumé
// d'un seuil contourné. Chaque raison doit dire par quel AUTRE niveau de test le
// code est censé être couvert — ou avouer qu'aucun ne le couvre.
type exclusion struct {
	prefix string
	reason string
}

// Raisons partagées. Nommées plutôt que répétées : le jour où l'issue #37 est
// close, il n'y a qu'un endroit à corriger — et le linter refuse la répétition
// précisément pour éviter qu'une des cinq copies survive à l'oubli.
const (
	reasonNeedsPostgres = "exige une base Postgres — AUCUN test à ce jour, à aucun niveau (voir #37)"
	reasonNeedsRedis    = "exige un Redis — AUCUN test à ce jour, à aucun niveau (voir #37)"
)

//nolint:gochecknoglobals // table de configuration du programme, lue une fois
var exclusions = []exclusion{
	{
		prefix: "internal/core/audit/drivers/postgres/",
		reason: reasonNeedsPostgres,
	},
	{
		prefix: "internal/core/dynconf/drivers/postgres/",
		reason: reasonNeedsPostgres,
	},
	{
		prefix: "internal/core/idempotency/drivers/postgres/",
		reason: reasonNeedsPostgres,
	},
	{
		prefix: "internal/core/idempotency/drivers/redis/",
		reason: reasonNeedsRedis,
	},
	{
		prefix: "internal/core/outbox/drivers/postgres/",
		reason: reasonNeedsPostgres,
	},
	{
		prefix: "internal/core/scheduler/drivers/postgres/",
		reason: reasonNeedsPostgres,
	},
	{
		prefix: "internal/infrastructure/database/",
		reason: "ouvre un pool pgx à la construction — exige une base (voir #37)",
	},
	{
		prefix: "internal/infrastructure/cache/",
		reason: "ouvre une connexion Redis à la construction — exige un service (voir #37)",
	},
	{
		prefix: "cmd/",
		reason: "composition root : exercé par tests/e2e (tag e2e), job CI e2e",
	},
	{
		prefix: "tools/",
		reason: "outillage de build, pas du code produit — s'inclure soi-même fausserait la mesure",
	},
}

// block est une plage de couverture lue dans le profil.
type block struct {
	path       string
	statements int
	covered    bool
}

func main() {
	if len(os.Args) < 2 {
		fail("usage: covergate <coverage.out>")
	}
	blocks, err := parseProfile(os.Args[1])
	if err != nil {
		fail(err.Error())
	}
	if len(blocks) == 0 {
		// Piège du faux vert : un profil vide n'est pas « tout couvert ».
		fail("profil de couverture VIDE — la commande de test n'a rien mesuré")
	}
	os.Exit(report(blocks))
}

// report applique les trois cliquets et rend le code de sortie.
func report(blocks []block) int {
	failed := false

	if err := checkExclusionsAreAlive(blocks); err != nil {
		fmt.Fprintf(os.Stderr, "\n  ÉCHEC  %s\n", err)
		failed = true
	}

	product := rate(blocks, isProduct)
	unit := rate(blocks, func(b block) bool { return !isExcluded(b.path) })
	core := rate(blocks, isCore)

	fmt.Println("┌─ Couverture ────────────────────────────────────────────────")
	printGate("périmètre unitaire", unit, unitThreshold, &failed)
	printGate("cœur (domain + application)", core, coreThreshold, &failed)
	printGate("code produit (cliquet)", product, productFloor, &failed)
	fmt.Println("└─────────────────────────────────────────────────────────────")

	printExcluded(blocks)

	if failed {
		return 1
	}
	return 0
}

// printGate affiche un cliquet et met `failed` à vrai s'il ne passe pas.
func printGate(name string, got, threshold float64, failed *bool) {
	verdict := "PASS"
	if got < threshold {
		verdict = "ÉCHEC"
		*failed = true
	}
	fmt.Printf("│ %-28s %5.1f %%   seuil %4.1f %%   %s\n", name, got, threshold, verdict)
}

// printExcluded rend visible ce qui est SORTI du périmètre unitaire, avec sa
// couverture réelle et sa raison.
//
// C'est la partie qui empêche l'exclusion d'être une dissimulation : elle
// s'imprime à CHAQUE exécution, en local comme en CI. Un périmètre réduit qu'on
// n'affiche pas se lit « on couvre tout ».
func printExcluded(blocks []block) {
	fmt.Println("\nHors périmètre unitaire — ces lignes ne sont PAS couvertes par `go test ./...` :")

	for _, exc := range exclusions {
		matched := filter(blocks, func(b block) bool { return strings.HasPrefix(b.path, exc.prefix) })
		stmts := 0
		for _, b := range matched {
			stmts += b.statements
		}
		fmt.Printf("  %-46s %5.1f %%  %4d instr.  %s\n",
			exc.prefix, rate(matched, func(block) bool { return true }), stmts, exc.reason)
	}
}

// checkExclusionsAreAlive refuse une entrée qui ne correspond plus à rien.
//
// Sans cette garde, la liste POURRIT : un paquet renommé laisse son ancienne
// entrée en place, et le périmètre s'élargit ou se rétrécit sans que personne
// l'ait décidé. Une exclusion morte est une exclusion qu'on ne relit plus.
func checkExclusionsAreAlive(blocks []block) error {
	var dead []string
	for _, exc := range exclusions {
		if !contains(blocks, func(b block) bool { return strings.HasPrefix(b.path, exc.prefix) }) {
			dead = append(dead, exc.prefix)
		}
	}
	if len(dead) == 0 {
		return nil
	}
	sort.Strings(dead)
	return fmt.Errorf(
		"exclusion(s) ne correspondant à aucun code mesuré: %s — "+
			"chemin renommé ou supprimé ? retirer l'entrée de tools/covergate",
		strings.Join(dead, ", "))
}

// isCore reconnaît le cœur métier : les règles, celles qui doivent tenir à 90 %.
func isCore(b block) bool {
	return strings.Contains(b.path, "/domain/") || strings.Contains(b.path, "/application/")
}

// isProduct reconnaît le code livré, par opposition à l'outillage de build.
func isProduct(b block) bool {
	return !strings.HasPrefix(b.path, buildTooling)
}

func isExcluded(path string) bool {
	for _, exc := range exclusions {
		if strings.HasPrefix(path, exc.prefix) {
			return true
		}
	}
	return false
}

// rate calcule la couverture PONDÉRÉE PAR INSTRUCTION sur les blocs retenus.
//
// Pondérée, et c'est un correctif : le cliquet du cœur moyennait les pourcentages
// PAR FONCTION. Une fonction d'une instruction pesait alors autant qu'une
// fonction de cinquante, si bien qu'une poignée de constructeurs triviaux
// couverts pouvait masquer une règle métier entière non testée.
func rate(blocks []block, keep func(block) bool) float64 {
	total, covered := 0, 0
	for _, b := range blocks {
		if !keep(b) {
			continue
		}
		total += b.statements
		if b.covered {
			covered += b.statements
		}
	}
	if total == 0 {
		return 0
	}
	return 100 * float64(covered) / float64(total)
}

func filter(blocks []block, keep func(block) bool) []block {
	out := make([]block, 0, len(blocks))
	for _, b := range blocks {
		if keep(b) {
			out = append(out, b)
		}
	}
	return out
}

func contains(blocks []block, match func(block) bool) bool {
	for _, b := range blocks {
		if match(b) {
			return true
		}
	}
	return false
}

// parseProfile lit un profil produit par `go test -coverprofile`.
//
// Format d'une ligne, après l'en-tête `mode:` :
//
//	<chemin>:<ligne>.<col>,<ligne>.<col> <nb instructions> <nb exécutions>
func parseProfile(path string) ([]block, error) {
	file, err := os.Open(path) //nolint:gosec // chemin fourni par la CI ou le Taskfile, pas par un tiers
	if err != nil {
		return nil, fmt.Errorf("lecture du profil: %w", err)
	}
	defer func() { _ = file.Close() }()

	// FUSION OBLIGATOIRE, et c'est le piège central de ce format.
	//
	// Avec `-coverpkg=./...`, CHAQUE binaire de test émet un profil pour TOUS
	// les paquets du dépôt. Une même plage apparaît donc autant de fois qu'il y
	// a de paquets de test — une vingtaine ici — et la quasi-totalité de ces
	// occurrences porte un compteur à zéro, puisqu'un binaire donné n'exerce
	// qu'une petite partie du dépôt.
	//
	// Les additionner naïvement écrase la mesure : 3,4 % au lieu de 56,9 %, et
	// des totaux d'instructions absurdes. C'est la MÊME faute, dans le MÊME
	// sens, que l'oubli de `-coverpkg` corrigé juste avant : un chiffre
	// faussement bas fait échouer le cliquet pour une raison inexistante, et
	// quelqu'un finit par baisser le seuil.
	//
	// La plage est donc la clé, et « couverte » est un OU sur toutes ses
	// occurrences : il suffit qu'UN binaire l'ait exécutée.
	merged := make(map[string]block)
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "mode:") {
			continue
		}
		key, parsed, err := parseLine(raw)
		if err != nil {
			return nil, fmt.Errorf("profil ligne %d: %w", line, err)
		}
		if seen, found := merged[key]; found {
			parsed.covered = parsed.covered || seen.covered
		}
		merged[key] = parsed
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parcours du profil: %w", err)
	}

	blocks := make([]block, 0, len(merged))
	for _, b := range merged {
		blocks = append(blocks, b)
	}
	return blocks, nil
}

// parseLine rend la CLÉ DE FUSION (le fichier et la plage, tels quels) et la
// plage lue.
func parseLine(raw string) (string, block, error) {
	fields := strings.Fields(raw)
	const expectedFields = 3
	if len(fields) != expectedFields {
		return "", block{}, fmt.Errorf("%d champs, %d attendus: %q", len(fields), expectedFields, raw)
	}
	statements, err := strconv.Atoi(fields[1])
	if err != nil {
		return "", block{}, fmt.Errorf("nombre d'instructions illisible: %q", fields[1])
	}
	count, err := strconv.Atoi(fields[2])
	if err != nil {
		return "", block{}, fmt.Errorf("nombre d'exécutions illisible: %q", fields[2])
	}
	return fields[0], block{
		path:       modulePath(fields[0]),
		statements: statements,
		covered:    count > 0,
	}, nil
}

// modulePath réduit le chemin du profil à un chemin relatif au dépôt.
//
// Le profil nomme les fichiers par leur chemin d'IMPORT complet
// (`github.com/…/internal/config/config.go:12.5,14.3`). Les exclusions, elles,
// sont écrites comme des chemins du dépôt : c'est ce qu'un relecteur reconnaît.
func modulePath(raw string) string {
	path := raw
	if colon := strings.LastIndex(path, ":"); colon >= 0 {
		path = path[:colon]
	}
	const marker = "go-hexa-fp-starter/"
	if idx := strings.Index(path, marker); idx >= 0 {
		return path[idx+len(marker):]
	}
	return path
}

func fail(message string) {
	fmt.Fprintf(os.Stderr, "covergate: %s\n", message)
	os.Exit(1)
}
