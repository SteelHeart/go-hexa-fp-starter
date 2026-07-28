package generator

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// featureTemplates porte l'anatomie d'un module métier.
//
// Embarquée plutôt que lue sur le disque : le binaire `hexa` doit fonctionner
// depuis n'importe où, y compris installé hors du dépôt.
//
// Les fichiers portent le suffixe `.tmpl` pour une raison mécanique : nommés
// `.go`, ils seraient compilés avec le reste du paquet, et un gabarit ne compile
// pas — il contient `{{.Module}}`.
//
//go:embed all:templates/feature
var featureTemplates embed.FS

// featureTemplateRoot est le préfixe à retirer des chemins embarqués.
const featureTemplateRoot = "templates/feature"

// CreateFeature rend le gabarit, durcit l'architecture, puis ÉPROUVE.
func CreateFeature(ctx context.Context, p FeaturePlan) error {
	if err := RenderFeature(p); err != nil {
		return err
	}
	if err := DeclareIsolation(p.anchor, p.Dir); err != nil {
		return err
	}
	return proveFeature(ctx, p)
}

// RenderFeature écrit l'arborescence du module.
func RenderFeature(p FeaturePlan) error {
	walk := fs.WalkDir(featureTemplates, featureTemplateRoot,
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			relative := strings.TrimPrefix(strings.TrimPrefix(path, featureTemplateRoot), "/")
			target := filepath.Join(p.Destination, strings.TrimSuffix(relative, ".tmpl"))
			return renderOne(path, target, p)
		})
	if walk != nil {
		return fmt.Errorf("rendu du gabarit de module: %w", walk)
	}
	return nil
}

// renderOne écrit un fichier du gabarit, formaté.
func renderOne(path, target string, p FeaturePlan) error {
	rendered, err := render(path, p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return fmt.Errorf("création de %s: %w", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, rendered, 0o600); err != nil {
		return fmt.Errorf("écriture de %s: %w", target, err)
	}
	return nil
}

// render applique le gabarit, puis FORMATE le résultat s'il est en Go.
//
// # Pourquoi le formatage est ici et non dans les gabarits
//
// Un gabarit ne peut pas être `gofmt`-propre par construction : la largeur des
// identifiants change avec le nom du module, donc tout alignement écrit à la main
// est faux pour tous les noms sauf un. Mesuré — la première version produisait
// trois fichiers que `go fmt` réécrivait, et l'étape `fmt` de la barrière du
// projet généré les signalait.
//
// `format.Source` supprime la classe entière : aucun futur remaniement d'un
// gabarit ne peut plus produire du code mal formaté. Et un gabarit devenu
// syntaxiquement faux ne s'écrit plus en silence — il refuse ici, avec sa
// position, au lieu d'échouer plus tard sur un `go build` du projet.
func render(path string, p FeaturePlan) ([]byte, error) {
	raw, err := fs.ReadFile(featureTemplates, path)
	if err != nil {
		return nil, fmt.Errorf("lecture du gabarit %s: %w", path, err)
	}
	// `missingkey=error` : un gabarit qui référencerait une clé absente écrirait
	// `<no value>` dans du code Go. Mieux vaut refuser bruyamment.
	model, err := template.New(path).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("gabarit %s illisible: %w", path, err)
	}

	var buffer bytes.Buffer
	if rendered := model.Execute(&buffer, p); rendered != nil {
		return nil, fmt.Errorf("rendu de %s: %w", path, rendered)
	}
	if !strings.HasSuffix(path, ".go.tmpl") {
		return buffer.Bytes(), nil
	}

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("le gabarit %s ne produit pas du Go valide: %w", path, err)
	}
	return formatted, nil
}

// proveFeature compile le projet ENTIER et exécute TOUS ses tests.
//
// La vérification fait partie de la commande, elle n'est pas laissée à
// l'utilisateur : un module généré qui ne compile pas doit le dire lui-même.
//
// Le projet entier plutôt que les seuls paquets du module neuf, pour deux
// raisons. La première est de fond : un module qui casse le reste du projet est
// un module fautif, et ne le constater qu'au `task check` suivant ferait porter
// le doute sur la mauvaise modification. La seconde est mécanique — `gosec`
// refuse à juste titre une commande dont un argument est une variable, et une
// dérogation motivée coûterait plus cher que la vérification plus large.
func proveFeature(ctx context.Context, p FeaturePlan) error {
	steps := []*exec.Cmd{
		exec.CommandContext(ctx, "go", "build", "./..."),
		exec.CommandContext(ctx, "go", "test", "./..."),
	}
	for _, cmd := range steps {
		cmd.Dir = p.Root
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("le module généré ne passe pas `%s` — la génération est "+
				"fautive, pas le module :\n%s", strings.Join(cmd.Args, " "), out)
		}
	}
	return nil
}
