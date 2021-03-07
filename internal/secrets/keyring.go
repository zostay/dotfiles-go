package secrets

import (
	"github.com/zalando/go-keyring"
)

const (
	SecretServiceName = "zostay-dotfiles" // the service to use with the system keyring service
)

// Keyring is a Keeper that allows the user to get and set secrets in the system
// keyring identified by SecretServiceName.
type Keyring struct{}

// GetSecret retrieves the named secret from the system keyring.
func (Keyring) GetSecret(name string) (string, error) {
	s, err := keyring.Get(SecretServiceName, name)
	if err != nil {
		return "", ErrNotFound
	}
	return s, nil
}

// SetSecret sets the named secret to the given value in the system keyring.
func (Keyring) SetSecret(name, secret string) error {
	return keyring.Set(SecretServiceName, name, secret)
}
