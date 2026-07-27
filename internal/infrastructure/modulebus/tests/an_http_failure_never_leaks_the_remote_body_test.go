package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// TestAnHTTPFailureNeverLeaksTheRemoteBody : un échec distant reste un statut.
//
// # Le défaut que ce test attrape — et c'est une fuite de données
//
// Le réflexe, en déboguant un appel inter-module, est de recopier le corps de la
// réponse distante dans le message d'erreur : « ça aide à diagnostiquer ». Ce
// corps est le message d'erreur d'UN AUTRE module, sur SES données. Il contient
// couramment l'adresse électronique refusée, l'identifiant du client, la raison
// métier du rejet.
//
// Cette erreur remonte à l'appelant, est journalisée par lui, et part vers le
// puits d'observabilité. Des données personnelles d'un module traversent alors la
// frontière d'un autre et atterrissent dans les journaux — exactement ce que
// rules/securite.md 5 interdit, et par le chemin le plus banal qui soit.
//
// La règle est donc : le statut, le point d'entrée, rien d'autre. Un module
// appelant n'a pas à connaître la taxonomie d'erreurs interne d'un autre ; il
// traduit lui-même à partir du statut.
func TestAnHTTPFailureNeverLeaksTheRemoteBody(t *testing.T) {
	t.Parallel()

	// Un corps qui ressemble à ce qu'un vrai module rendrait sur un 422.
	const secretish = `{"detail":"adresse alice@example.com deja enregistree","customer":"cus_00042"}`

	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusUnprocessableEntity,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	} {
		remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(secretish))
		}))

		var localCalls int
		cfg := interop("http", map[string]string{someModule: remote.URL})
		call := resolve(t, cfg, noPublisher(t), localCaller(&localCalls))

		_, err := call(context.Background(), request{Ref: "r-1"})
		remote.Close()

		if err == nil {
			t.Errorf("statut %d accepté comme un succès", status)
			continue
		}
		message := err.Error()

		if strings.Contains(message, "alice@example.com") ||
			strings.Contains(message, "cus_00042") ||
			strings.Contains(message, "deja enregistree") {
			t.Errorf("statut %d: le corps distant a fuité dans l'erreur: %s", status, message)
		}
		// Le statut doit y être : sans lui, l'appelant ne peut pas traduire.
		if !strings.Contains(message, strconv.Itoa(status)) {
			t.Errorf("statut %d absent du message %q — l'appelant ne peut pas traduire", status, message)
		}
	}
}

// TestAnUnreadableReplyIsAFailure : un corps illisible n'est pas un succès.
//
// Un 200 dont le corps n'est pas le JSON attendu — module de la mauvaise version,
// mandataire qui insère une page d'erreur — doit échouer. Rendre la valeur zéro
// ferait lire « refusé » à l'appelant sur une réponse qu'il n'a jamais reçue.
func TestAnUnreadableReplyIsAFailure(t *testing.T) {
	t.Parallel()

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>502 Bad Gateway</html>"))
	}))
	defer remote.Close()

	var localCalls int
	cfg := interop("http", map[string]string{someModule: remote.URL})
	call := resolve(t, cfg, noPublisher(t), localCaller(&localCalls))

	if _, err := call(context.Background(), request{Ref: "r-1"}); err == nil {
		t.Error("une réponse illisible a été acceptée — l'appelant lirait la valeur zéro")
	}
}
