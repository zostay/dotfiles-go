package secrets

import (
	"fmt"
	"os"
)

// Cacher is a dual component secret Keeper that attempts to keep both of the
// keepers present in sync. One Keeper is treated as the source of truth and the
// other is the target for more truth.
type Cacher struct {
	source Keeper // the source of truth
	target Keeper // the receiver of truth
}

// NewCacher constructs a Cacher Keeper from the given source and target
// Keepers.
func NewCacher(src, target Keeper) *Cacher {
	return &Cacher{src, target}
}

// GetSecret retrieves the requested secret from the target Keeper. If it is not
// found on the target, it retreives it from the source. An error is returned if
// this retrieval fails (including failure of ErrNotFound). Hoewver, if the get
// succeeds, the target is updated to set the secret in the target Keeper.
//
// If the initial get from the target results in an error other than
// ErrNotFound, that error is returned with no other action having been
// performed.
//
// If the initial get from the target succeeds, the result is returned
// immediately.
func (c *Cacher) GetSecret(name string) (string, error) {
	s, err := c.target.GetSecret(name)
	if err == ErrNotFound {
		s, err := c.source.GetSecret(name)
		if err != nil {
			return s, err
		}

		err = c.target.SetSecret(name, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error caching secret %s locally: %v\n", name, err)
			return s, nil
		}
	} else if err != nil {
		return "", err
	}

	return s, nil
}

// SetSecret sets the secret in both the source and target Keepers.
func (c *Cacher) SetSecret(name, secret string) error {
	err1 := c.target.SetSecret(name, secret)
	err2 := c.source.SetSecret(name, secret)

	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	}

	return nil
}
