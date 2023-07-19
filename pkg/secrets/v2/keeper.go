package secrets

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("secret not found") // error returned by a secrets.Keeper when a secret is not found
)

// Keeper is a tool for storing and retrieving secrets. Locations are treated as
// flat opaque refs as far as this API is concerned, however, individual Keepers
// may provide multi-level hierarchies of locations.
//
// This assumes a given secret is only found in a single location. The ID field
// must be unique throughout the storage and must be assigned whenever a Secret
// is returned by one of these methods. The Name field is not guaranteed to be
// unique. If the Location field is unset, this indicates the secret is to be
// stored in the default location. The Fields lists all those fields that do not
// have their own accessor.
//
// Even if a Keeper storage uses one list of properties to store all, the fields
// with their own accessor should not be returned by Fields or GetField.
type Keeper interface {
	// ListLocations returns the names of every storage location.
	ListLocations(ctx context.Context) ([]string, error)

	// ListSecrets returns the name of the secrets stored at the given location.
	ListSecrets(ctx context.Context, location string) ([]string, error)

	// GetSecretsByName returns all secrets stored with that name. This should
	// not return the ErrNotFound error if no secret with the given name is
	// found.
	GetSecretsByName(ctx context.Context, name string) ([]Secret, error)

	// GetSecret returns a secret by unique ID, which is Keeper dependant. If no
	// secret is found for the given ID, this function should returned a nil
	// Secret with ErrNotFound.
	GetSecret(ctx context.Context, id string) (Secret, error)

	// SetSecret performs an insertion or update of the secret. If the secret
	// has a valid ID that matches a record in Keeper storage, it will update
	// that secret in the store. If the ID is not valid or not found in Keeper
	// storage, a new value will be inserted.
	//
	// In either case, a new Secret object will be returned. The old value
	// should now be considered invalid.
	SetSecret(ctx context.Context, secret Secret) (Secret, error)

	// CopySecret copies a secret to a new location while keeping the secret in
	// the existing location as well. The secret passed in remains valid. A new
	// secret representing the newly made copy is returned.
	CopySecret(ctx context.Context, secret Secret, location string) (Secret, error)

	// MoveSecret moves a secret to a new location. The passed in secret is
	// invalid after this call is made. The newly moved secret object is
	// returned.
	MoveSecret(ctx context.Context, secret Secret, location string) (Secret, error)

	// DeleteSecret removes the secret.
	DeleteSecret(ctx context.Context, id string) error
}
