package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration est une durée lisible dans un fichier de configuration.
//
// # Pourquoi ce type existe
//
// `gopkg.in/yaml.v3` ne décode PAS une chaîne comme "5s" en time.Duration : il
// n'accepte qu'un entier de nanosecondes. Sans ce type, toute la configuration
// échouerait au chargement — et le message d'erreur ne dirait pas pourquoi.
//
// Le type sous-jacent EST time.Duration, donc la conversion au site d'usage est
// gratuite : `time.Duration(cfg.HTTP.ReadTimeout)`.
type Duration time.Duration

// UnmarshalYAML accepte les deux formes utiles :
//   - une chaîne : "5s", "1h30m", "250ms" — la forme attendue dans conf
//   - un entier  : interprété en SECONDES, jamais en nanosecondes
//
// Les nanosecondes sont volontairement refusées : `read_timeout: 5` doit valoir
// cinq secondes, pas cinq nanosecondes. Interpréter un entier nu en nanosecondes
// produirait un délai de zéro en pratique, donc une panne silencieuse.
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	// Discriminer sur le TAG, pas sur le succès du décodage : yaml.v3 décode
	// volontiers un nœud entier vers une string, donc tenter la chaîne d'abord
	// ferait échouer `30` au lieu de le lire comme trente secondes.
	if node.Tag == "!!int" {
		var asSeconds int64
		if err := node.Decode(&asSeconds); err != nil {
			return fmt.Errorf("durée entière illisible à la ligne %d: %w", node.Line, err)
		}
		*d = Duration(time.Duration(asSeconds) * time.Second)
		return nil
	}

	var asString string
	if err := node.Decode(&asString); err != nil {
		return fmt.Errorf(
			"durée illisible à la ligne %d (attendu une chaîne comme \"5s\") : %w", node.Line, err)
	}
	parsed, err := time.ParseDuration(asString)
	if err != nil {
		return fmt.Errorf("durée %q illisible (attendu: 5s, 1h30m, 250ms): %w", asString, err)
	}
	*d = Duration(parsed)
	return nil
}

// MarshalYAML réécrit la durée sous sa forme lisible, pour que la configuration
// effective puisse être affichée telle qu'elle serait écrite.
func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

// Duration convertit vers le type de la bibliothèque standard.
func (d Duration) Duration() time.Duration { return time.Duration(d) }

// String rend la forme lisible.
func (d Duration) String() string { return time.Duration(d).String() }
