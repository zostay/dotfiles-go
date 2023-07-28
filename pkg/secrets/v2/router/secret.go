package router

import (
	"strings"

	"github.com/zostay/dotfiles-go/pkg/secrets/v2"
)

// Secret is a special wrapper around secrets returned from Router that manages
// the ID.
type Secret struct {
	secrets.Secret
	id string
}

var _ secrets.Secret = &Secret{}

// makeId creates an ID from a keeper ID and a secret ID.
func makeId(keeperId, secretId string) string {
	return strings.Join([]string{keeperId, secretId}, ":")
}

// newSecret creates a new secret combined with the given keeper ID.
func newSecret(keeperId string, secret secrets.Secret) *Secret {
	return &Secret{
		Secret: secret,
		id:     makeId(keeperId, secret.ID()),
	}
}

// ID returns the ID of the secret.
func (s *Secret) ID() string {
	return s.id
}
