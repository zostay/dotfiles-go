package secrets

import (
	"errors"
)

// LocumTenens is a Keeper that stands in the place of others. It wraps zero or
// more other Keepers. Secrets gotten from it will return the first secret
// found. Secrets stored to it will store in the first Keeper that does not
// return an error when storing the secret.
type LocumTenens struct {
	keepers []Keeper // the list of keepers
}

// NewLocumTenens constructs a new LocumTenens. Use AddKeeper to add Keepers
// inside before using it. If you do not, GetSecret will always return
// ErrNotFound and SetSecret will always fail with an error.
func NewLocumTenens() *LocumTenens {
	return &LocumTenens{
		keepers: make([]Keeper, 0, 2),
	}
}

// AddKeeper adds the given Keeper to those wrapped. GetSecret and SetSecret
// operations will prefer Keepers added first.
func (l *LocumTenens) AddKeeper(k Keeper) {
	l.keepers = append(l.keepers, k)
}

// GetSecret returns the first secret found by querying each wrapped Keeper. If
// no keepers are wrapped or the secret is found in none of them, it returns
// ErrNotFound.
func (l *LocumTenens) GetSecret(name string) (*Secret, error) {
	for _, k := range l.keepers {
		s, err := k.GetSecret(name)
		if err == nil {
			return s, nil
		}
	}
	return nil, ErrNotFound
}

// SetSecret tries to store the secret in each Keeper in the ordered they were
// added via calls to AddKeeper. If SetSecret for a keeper returns an error, the
// next keeper is tried until there's a success. Then the operation quits. If
// there are zero Keepers or they all return errors, then this returns an error
// as well.
func (l *LocumTenens) SetSecret(secret *Secret) error {
	for _, k := range l.keepers {
		err := k.SetSecret(secret)
		if err == nil {
			return nil
		}
	}
	return errors.New("no secret keeper able to store secret")
}

// RemoveSecret removes the secret from each keeper.
func (l *LocumTenens) RemoveSecret(name string) error {
	var ferr error
	for _, k := range l.keepers {
		err := k.RemoveSecret(name)
		if err != nil && ferr == nil {
			ferr = err
		}
	}

	return ferr
}
