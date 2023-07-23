package seq

import (
	"context"
	"errors"

	"github.com/zostay/go-std/set"

	"github.com/zostay/dotfiles-go/pkg/secrets/v2"
)

// Seq is a Keeper that gets secrets from the first Keeper that returns it.
type Seq struct {
	keepers []secrets.Keeper
}

var _ secrets.Keeper = &Seq{}

// Returns a new sequential keeper with the given list of Keepers.
func NewSeq(keepers ...secrets.Keeper) (*Seq, error) {
	if len(keepers) == 0 {
		return nil, errors.New("no keepers given")
	}

	return &Seq{
		keepers: keepers,
	}, nil
}

// ListLocations returns the list of locations from all Keepers.
func (s *Seq) ListLocations(ctx context.Context) ([]string, error) {
	locations := set.New[string]()
	for _, k := range s.keepers {
		locs, err := k.ListLocations(ctx)
		if err != nil {
			return nil, err
		}
		for _, loc := range locs {
			locations.Insert(loc)
		}
	}
	return locations.Keys(), nil
}

// ListSecrets returns the list of secrets from all Keepers in the named
// location.
func (s *Seq) ListSecrets(
	ctx context.Context,
	location string,
) ([]string, error) {
	secrets := set.New[string]()
	for _, k := range s.keepers {
		secs, err := k.ListSecrets(ctx, location)
		if err != nil {
			return nil, err
		}
		for _, sec := range secs {
			secrets.Insert(sec)
		}
	}
	return secrets.Keys(), nil
}

// GetSecret returns the secret from the first Keeper that returns it.
func (s *Seq) GetSecret(
	ctx context.Context,
	id string,
) (secrets.Secret, error) {
	for _, k := range s.keepers {
		sec, err := k.GetSecret(ctx, id)
		if err != nil {
			if !errors.Is(err, secrets.ErrNotFound) {
				return nil, err
			}
			continue
		}
		return sec, nil
	}
	return nil, secrets.ErrNotFound
}

// GetSecretsByName returns all secrets with the given name from all Keepers.
func (s *Seq) GetSecretsByName(
	ctx context.Context,
	name string,
) ([]secrets.Secret, error) {
	secrets := make([]secrets.Secret, 0, 1)
	for _, k := range s.keepers {
		secs, err := k.GetSecretsByName(ctx, name)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, secs...)
	}
	return secrets, nil
}

// SetSecret stores the secret in the first Keeper.
func (s *Seq) SetSecret(
	ctx context.Context,
	sec secrets.Secret,
) (secrets.Secret, error) {
	return s.keepers[0].SetSecret(ctx, sec)
}

// DeleteSecret deletes the secret from all Keepers.
func (s *Seq) DeleteSecret(
	ctx context.Context,
	id string,
) error {
	for _, k := range s.keepers {
		err := k.DeleteSecret(ctx, id)
		if err != nil {
			return err
		}
	}
	return nil
}

// CopySecret gets the secret from any of the Keepers in the Seq and makes a
// copy of it in the first Keeper at the new location.
func (s *Seq) CopySecret(
	ctx context.Context,
	id string,
	location string,
) (secrets.Secret, error) {
	sec, err := s.GetSecret(ctx, id)
	if err != nil {
		return nil, err
	}

	newSec := secrets.NewSingleFromSecret(sec,
		secrets.WithID(""),
		secrets.WithLocation(location))
	return s.SetSecret(ctx, newSec)
}

// MoveSecret gets the secret from any of the Keepers in the Seq, then deletes
// it from all Keepers, and then stores it in the first Keeper at the new
// location.
func (s *Seq) MoveSecret(
	ctx context.Context,
	id string,
	location string,
) (secrets.Secret, error) {
	sec, err := s.GetSecret(ctx, id)
	if err != nil {
		return nil, err
	}

	s.DeleteSecret(ctx, id)

	newSec := secrets.NewSingleFromSecret(sec,
		secrets.WithID(""),
		secrets.WithLocation(location))
	return s.SetSecret(ctx, newSec)
}
