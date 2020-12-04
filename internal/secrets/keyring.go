package secrets

import (
	"github.com/zalando/go-keyring"
)

const (
	SecretServiceName = "zostay-dotfiles"
)

type Keyring struct{}

func (Keyring) GetSecret(name string) (string, error) {
	s, err := keyring.Get(SecretServiceName, name)
	if err != nil {
		return "", ErrNotFound
	}
	return s, nil
}

func (Keyring) SetSecret(name, secret string) error {
	return keyring.Set(SecretServiceName, name, secret)
}
