package secrets

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ansd/lastpass-go"
)

// LastPass is a secret Keeper that gets secrets from the LastPass
// password manager service.
type LastPass struct {
	lp    *lastpass.Client
	cat   string
	limit bool
}

// NewLastPass constructs and returns a new LastPass Keeper or returns an error
// if there was a problem during construction.
//
// The cat argument sets the name of the group to use when setting secrets. If
// the limit parameter is true, then getting a secret will be limited to secrets
// in the group named by cat.
func NewLastPass(cat string, limit bool) (*LastPass, error) {
	u := LastPassUsername
	if LastPassUsername == "" {
		var err error
		u, err = PinEntry(
			"Zostay LastPass",
			"Asking for LastPass Username",
			"Username:",
			"OK",
		)
		if err != nil {
			return nil, err
		}
	}

	p, err := GetMasterPassword("LastPass", "LASTPASS-MASTER-"+u)
	if err != nil {
		return nil, err
	}

	lp, err := lastpass.NewClient(context.Background(), u, p)
	if err != nil {
		return nil, err
	}

	err = SetMasterPassword("LASTPASS-MASTER-"+u, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error keeping master password in memory.")
	}

	return &LastPass{lp, cat, limit}, nil
}

// GetSecret returns the secret from the Lastpass service.
func (l *LastPass) GetSecret(name string) (*Secret, error) {
	as, err := l.lp.Accounts(context.Background())
	if err != nil {
		return nil, err
	}

	for _, a := range as {
		if l.limit && a.Group != l.cat {
			continue
		}

		if a.Name == name {
			return &Secret{
				Name:  name,
				Value: a.Password,
			}, nil
		}
	}

	return nil, ErrNotFound
}

// SetSecret sets the secret into the LastPass service.
func (l *LastPass) SetSecret(secret *Secret) error {
	as, err := l.lp.Accounts(context.Background())
	if err != nil {
		return err
	}

	for _, a := range as {
		if a.Name == secret.Name {
			a.Password = secret.Value
			err := l.lp.Update(context.Background(), a)
			return err
		}
	}

	a := lastpass.Account{
		Name:     secret.Name,
		Password: secret.Value,
		Group:    l.cat,
	}

	err = l.lp.Add(context.Background(), &a)
	return err
}

// RemoveSecret removes the secret from the LastPass service.
func (l *LastPass) RemoveSecret(name string) error {
	return errors.New("not implemented")
}
