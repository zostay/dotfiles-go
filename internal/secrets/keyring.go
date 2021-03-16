package secrets

import (
	"github.com/zalando/go-keyring"
)

// Keyring is a Keeper that allows the user to get and set secrets in the system
// keyring identified by SecretServiceName.
type Keyring struct {
	ssn string
}

// NewKeyring constructs a new secret Keeper that can talkt ot he system
// keyring tools. You must specify a service name to identify the application
// with.
func NewKeyring(ssn string) *Keyring {
	return &Keyring{ssn}
}

// GetSecret retrieves the named secret from the system keyring.
func (k *Keyring) GetSecret(name string) (*Secret, error) {
	s, err := keyring.Get(k.ssn, name)
	if err != nil {
		return nil, ErrNotFound
	}
	return &Secret{
		Name:  name,
		Value: s,
	}, nil
}

// SetSecret sets the named secret to the given value in the system keyring.
func (k *Keyring) SetSecret(secret *Secret) error {
	return keyring.Set(k.ssn, secret.Name, secret.Value)
}

// RemoveSecret deletes the named secret.
func (k *Keyring) RemoveSecret(name string) error {
	return keyring.Delete(k.ssn, name)
}
