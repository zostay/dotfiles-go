package router

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
	"github.com/zostay/go-std/set"
	"github.com/zostay/go-std/slices"

	"github.com/zostay/dotfiles-go/pkg/secrets/v2"
)

type keeperMap struct {
	id        string
	keeper    secrets.Keeper
	locations set.Set[string]
}

// Router is a Keeper that maps locations to other Keepers. Keepers are added to
// the Router using the AddKeeper method.
//
// The added secrets.Keepers are associated with this secrets.Keeper operate as
// peers, rather than children. That is, if you add another keeper for locations
// "Personal" and "Work", then whenever a secret is gotten or saved for
// "Personal" or for "Work", that keeper will be used. This is NOT a
// parent-child relationship with pathing or anything of that sort.
//
// Whenever secrets are fetched, saved, etc. The location of the secret will
// match those associated with a Keeper. For example, if GetSecretsByName is
// called, secrets that match that name, but are not in an associated location
// will not be returned.
//
// The IDs used by this library will differ from those returned by each
// associated keeper.
type Router struct {
	defaultId     string
	defaultKeeper secrets.Keeper
	keepers       []keeperMap

	usedLocations set.Set[string]
}

var _ secrets.Keeper = &Router{}

// NewRouter returns a new router with the given Keeper as the default Keeper.
func NewRouter(defaultKeeper secrets.Keeper) *Router {
	return &Router{
		defaultId:     ulid.Make().String(),
		defaultKeeper: defaultKeeper,
		keepers:       []keeperMap{},
		usedLocations: set.New[string](),
	}
}

// AddKeeper adds a new Keeper, which will be used for storing at the given
// locations. The same location may not be used for more than one Keeper.
func (r *Router) AddKeeper(keeper secrets.Keeper, locations ...string) error {
	newLocs := set.New(locations...)
	if set.Intersects(r.usedLocations, newLocs) {
		return fmt.Errorf("these locations are already in use: %s",
			strings.Join(
				set.Intersection(r.usedLocations, newLocs).Keys(),
				", "))
	}

	r.keepers = append(r.keepers, keeperMap{
		id:        ulid.Make().String(),
		keeper:    keeper,
		locations: set.New(locations...),
	})
	r.usedLocations = set.Union(r.usedLocations, newLocs)

	return nil
}

// ListLocations returns all the locations that this secrets.Keeper provides.
func (r *Router) ListLocations(ctx context.Context) ([]string, error) {
	locs, err := r.defaultKeeper.ListLocations(ctx)
	if err != nil {
		return nil, err
	}

	return set.Union(set.New(locs...), r.usedLocations).Keys(), nil
}

var errStop = errors.New("stop")

func handleStop(err error) error {
	if errors.Is(err, errStop) {
		return nil
	}
	return err
}

func (r *Router) keeperForLocation(
	ctx context.Context,
	location string,
) (string, secrets.Keeper) {
	for _, m := range r.keepers {
		if m.locations.Contains(location) {
			return m.id, m.keeper
		}
	}
	return r.defaultId, r.defaultKeeper
}

func (r *Router) defaultLocations(
	ctx context.Context,
) (set.Set[string], error) {
	defaultLocs, err := r.defaultKeeper.ListLocations(ctx)
	if err != nil {
		return nil, err
	}

	return set.Difference(set.New(defaultLocs...), r.usedLocations), nil
}

func (r *Router) forEachKeeperLocation(
	ctx context.Context,
	run func(string, secrets.Keeper, string) error,
) error {
	for _, m := range r.keepers {
		for _, loc := range m.locations.Keys() {
			err := run(m.id, m.keeper, loc)
			if err != nil {
				return err
			}
		}
	}

	defaultLocs, err := r.defaultLocations(ctx)
	if err != nil {
		return err
	}

	for _, loc := range defaultLocs.Keys() {
		err := run(r.defaultId, r.defaultKeeper, loc)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Router) forEachSecretInKeeperLocation(
	ctx context.Context,
	keeperId string,
	keeper secrets.Keeper,
	location string,
	run func(secrets.Secret) error,
) error {
	ids, err := keeper.ListSecrets(ctx, location)
	if err != nil {
		return err
	}

	for _, id := range ids {
		secret, err := keeper.GetSecret(ctx, id)
		if err != nil {
			return err
		}

		err = run(newSecret(keeperId, secret))
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Router) forEachSecret(
	ctx context.Context,
	run func(secrets.Secret) error,
) error {
	err := r.forEachKeeperLocation(ctx,
		func(id string, k secrets.Keeper, loc string) error {
			return r.forEachSecretInKeeperLocation(ctx, id, k, loc, run)
		},
	)

	return err
}

func (r *Router) findSecretMatchingId(
	ctx context.Context,
	id string,
) (string, secrets.Keeper, secrets.Secret, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", nil, nil, errors.New("bad secret ID in secret router")
	}

	keeperId, secretId := parts[0], parts[1]

	var (
		keeper    secrets.Keeper
		locations set.Set[string]
	)
	if r.defaultId == keeperId {
		var err error
		keeper = r.defaultKeeper
		locations, err = r.defaultLocations(ctx)
		if err != nil {
			return "", nil, nil, err
		}
	} else {
		for _, m := range r.keepers {
			if m.id == keeperId {
				keeper = m.keeper
				locations = m.locations
			}
		}
	}

	secret, err := keeper.GetSecret(ctx, secretId)
	if err != nil {
		return "", nil, nil, err
	}

	if !locations.Contains(secret.Location()) {
		return "", nil, nil, secrets.ErrNotFound
	}

	return keeperId, keeper, secret, nil
}

func (r *Router) forEachSecretInLocation(
	ctx context.Context,
	location string,
	run func(secret secrets.Secret) error,
) error {
	keeperId, keeper := r.keeperForLocation(ctx, location)
	return r.forEachSecretInKeeperLocation(ctx, keeperId, keeper, location, run)
}

// ListSecrets will list all secrets from the secrets.Keeper store that owns the
// given location.
func (r *Router) ListSecrets(
	ctx context.Context,
	location string,
) ([]string, error) {
	keeperId, keeper := r.keeperForLocation(ctx, location)
	ids, err := keeper.ListSecrets(ctx, location)
	if err != nil {
		return nil, err
	}
	return slices.Map(ids, func(id string) string {
		return makeId(keeperId, id)
	}), nil
}

// GetSecret will retrieve the identified secret from one of the available
// secrets.Keeper stores. This will return secrets.ErrNotFound if no secret
// matches the given ID.
func (r *Router) GetSecret(
	ctx context.Context,
	id string,
) (secrets.Secret, error) {
	_, _, secret, err := r.findSecretMatchingId(ctx, id)
	return secret, err
}

// GetSecretsByName will retrieve every secret in every secret store with the
// the given name.
func (r *Router) GetSecretsByName(
	ctx context.Context,
	name string,
) ([]secrets.Secret, error) {
	foundSecs := []secrets.Secret{}
	err := r.forEachSecret(ctx,
		func(secret secrets.Secret) error {
			if secret.Name() == name {
				foundSecs = append(foundSecs, secret)
			}
			return nil
		},
	)
	return foundSecs, handleStop(err)
}

// SetSecret will find the secrets.Keeper store used for the new secret's
// location. It will then add the secret to that Keeper.
func (r *Router) SetSecret(
	ctx context.Context,
	sec secrets.Secret,
) (secrets.Secret, error) {
	keeperId, keeper, secret, err := r.findSecretMatchingId(ctx, sec.ID())
	if err != nil {
		if errors.Is(err, secrets.ErrNotFound) {
			keeperId, keeper = r.keeperForLocation(ctx, sec.Location())
			newSec, err := keeper.SetSecret(ctx, sec)
			if err != nil {
				return nil, err
			}

			return newSecret(keeperId, newSec), nil
		}
		return nil, err
	}

	if secret.Location() != sec.Location() {
		return nil, errors.New("cannot move secret location with SetSecret in secret router")
	}

	newSec, err := keeper.SetSecret(ctx, sec)
	if err != nil {
		return nil, err
	}

	return newSecret(keeperId, newSec), nil
}

// DeleteSecret finds the store that holds the identified secret and deletes it.
func (r *Router) DeleteSecret(
	ctx context.Context,
	id string,
) error {
	_, keeper, _, err := r.findSecretMatchingId(ctx, id)
	if err != nil {
		return err
	}

	return keeper.DeleteSecret(ctx, id)
}

// CopySecret will copy the secret from one location to another, possibly moving
// it into another secrets.Keeper store.
//
// As of this writing, it will not use CopySecret even if a single
// secrets.Keeper store is used for both location. Instead, it copies the
// secrets.Password in memory with a blank ID and the new location and uses
// SetSecret to create it.
func (r *Router) CopySecret(
	ctx context.Context,
	id string,
	location string,
) (secrets.Secret, error) {
	keeperId, _, secret, err := r.findSecretMatchingId(ctx, id)
	if err != nil {
		return nil, err
	}

	if secret.Location() == location {
		return newSecret(keeperId, secret), nil
	}

	secret = secrets.NewSingleFromSecret(secret,
		secrets.WithID(""),
		secrets.WithLocation(location))
	keeperId, keeper := r.keeperForLocation(ctx, location)
	newSec, err := keeper.SetSecret(ctx, secret)
	if err != nil {
		return nil, err
	}

	return newSecret(keeperId, newSec), nil
}

// MoveSecret will move the secret from one location to another, possibly moving
// it into another secrets.Keeper store.
//
// As of this writing, this operation does not perform MoveSecret when the
// operation is moving between locations on the same store. Instead, this uses
// SetSecret to create a secret in the new location and then uses DeleteSecret
// to delete the secret from the old location. It is done in this order so that
// the accidental failure mode is more likely to end up with duplicated secrets
// than deleted secrets if this operation should fail in the middle.
func (r *Router) MoveSecret(
	ctx context.Context,
	id string,
	location string,
) (secrets.Secret, error) {
	keeperId, keeper, secret, err := r.findSecretMatchingId(ctx, id)
	if err != nil {
		return nil, err
	}

	if secret.Location() == location {
		return newSecret(keeperId, secret), nil
	}

	newKeeperId, newKeeper := r.keeperForLocation(ctx, location)
	newSingle := secrets.NewSingleFromSecret(secret,
		secrets.WithID(""),
		secrets.WithLocation(location))
	newSec, err := newKeeper.SetSecret(ctx, newSingle)

	err = keeper.DeleteSecret(ctx, id)
	return newSecret(newKeeperId, newSec), err
}
