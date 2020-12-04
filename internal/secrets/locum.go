package secrets

import (
	"errors"
)

type LocumTenens struct {
	keepers []Keeper
}

func NewLocumTenens() *LocumTenens {
	return &LocumTenens{
		keepers: make([]Keeper, 0, 2),
	}
}

func (l *LocumTenens) AddKeeper(k Keeper) {
	l.keepers = append(l.keepers, k)
}

func (l *LocumTenens) GetSecret(name string) (string, error) {
	for _, k := range l.keepers {
		s, err := k.GetSecret(name)
		if err == nil {
			return s, nil
		}
	}
	return "", ErrNotFound
}

func (l *LocumTenens) SetSecret(name, secret string) error {
	for _, k := range l.keepers {
		err := k.SetSecret(name, secret)
		if err == nil {
			return nil
		}
	}
	return errors.New("no secret keeper able to store secret")
}
