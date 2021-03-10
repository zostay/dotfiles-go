// Package secrets is a helper I use to store my secrets for use with my
// dotfiles but in such a way as that I don't store the secrets in the dotfiles.
package secrets

import "errors"

const (
	ZostayRobotGroup = "Robot" // category name for automatically managed secrets
)

var (
	ErrNotFound = errors.New("secret not found") // error returned by a secrets.Keeper when a secret is not found
	Master      = NewHttp()                      // the master password Keeper
	autoKeeper  Keeper                           // this is a local cache of secrets
)

// Keeper is the interface that all secret keepers follow.
type Keeper interface {
	// GetSecret should return the secret with the given name. If it makes a
	// difference to the storage mechanism, the storage should prefer secrets
	// found in the category named by ZostayRobotGroup.
	//
	// On success, return the secret string and no error.
	//
	// When the secret is not found, return an empty string and ErrNotFound.
	//
	// When their is an error with the secret store, return an empty string and
	// an error.
	GetSecret(name string) (string, error)

	// SetSecret stores the secret in the Keeper's store. The two arguments are
	// the name to give the secret and the cleartext secret, resepctively. For
	// stores where it matters, the secret should be stored in the group or
	// category named by ZostayRobotGroup.
	//
	// On success, this method should return nil.
	//
	// If there is a problem storing the secret, this method should return an error.
	SetSecret(name string, secret string) error
}

// AutoKeeper returns the Keeper preferred for password storage in the current
// application. It is lazily constructed on the first call to this function or
// can be set by the application via SetAutoKeeper.
func AutoKeeper() Keeper {
	setupBuiltinKeepers()

	return autoKeeper
}

// SetAutoKeeper replaces the local caching keeper with another.
func SetAutoKeeper(k Keeper) { autoKeeper = k }

// setupBuiltinKeepers is called lazily to setup the AutoKeeper when that
// function is called and SetAutoKeeper has not been called yet. This provides
// the default Keeper configuration for any application.
func setupBuiltinKeepers() {
	if autoKeeper != nil {
		return
	}

	kp, err1 := NewKeepass()
	lp, err2 := NewLastPass()
	if err2 == nil && err1 == nil {
		autoKeeper = NewCacher(lp, kp)
	} else if err1 == nil {
		autoKeeper = kp
	} else if err2 == nil {
		autoKeeper = lp
	} else {
		i, err := NewInternal()
		if err != nil {
			panic("unable to create any kind of secret keeper")
		}

		autoKeeper = i
	}
}
