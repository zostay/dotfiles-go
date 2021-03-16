package secrets

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Cacher is a dual component secret Keeper that attempts to keep both of the
// keepers present in sync. One Keeper is treated as the source of truth and the
// other is the target for more truth.
type Cacher struct {
	source Keeper // the source of truth
	target Keeper // the receiver of truth

	timeout time.Duration // secrets found in the target older than this will be resync'd
}

// NewCacher constructs a Cacher Keeper from the given source and target
// Keepers.
func NewCacher(src, target Keeper, timeout time.Duration) *Cacher {
	return &Cacher{src, target, timeout}
}

// sync performs the work of synchronizing the source and target for the given
// secret on get. Whatever state the source is in takes precedent.
func (c *Cacher) sync(name string) (*Secret, error) {
	s, err := c.source.GetSecret(name)
	if err == ErrNotFound {
		err := c.target.RemoveSecret(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error removing secret %q from cache: %v\n", name, err)
		}
		return nil, ErrNotFound
	} else if err != nil {
		return s, err
	}

	s.LastModified = time.Now()
	err = c.target.SetSecret(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error caching secret %q locally: %v\n", name, err)
		return s, nil
	}

	return s, nil
}

// GetSecret retrieves the requested secret from the target Keeper. If it is not
// found on the target, it retreives it from the source. An error is returned if
// this retrieval fails (including failure of ErrNotFound). Hoewver, if the get
// succeeds, the target is updated to set the secret in the target Keeper.
//
// If the secret is retreived from the target and the target has a non-zero
// LastModified time, that time is checked to see if it's older than the timeout
// configured during construction. If it is, the secret is retrieved from source
// anyway to resync.
//
// If the initial get from the target results in an error other than
// ErrNotFound, that error is returned with no other action having been
// performed.
//
// If the initial get from the target succeeds, the result is returned
// immediately.
func (c *Cacher) GetSecret(name string) (*Secret, error) {
	s, err := c.target.GetSecret(name)
	if err == nil {
		timeout := time.Now().Add(c.timeout)
		if !s.LastModified.IsZero() && s.LastModified.Before(timeout) {
			res, err := c.sync(name)
			if err != ErrNotFound {
				return nil, ErrNotFound
			} else if err != nil {
				return s, nil
			} else {
				return res, nil
			}
		}
	} else if err == ErrNotFound {
		s, err := c.sync(name)
		if err != nil {
			return nil, err
		}
		return s, nil
	} else {
		return nil, err
	}

	return s, nil
}

// SetSecret sets the secret in both the source and target Keepers.
func (c *Cacher) SetSecret(secret *Secret) error {
	err := c.source.SetSecret(secret)
	if err != nil {
		return err
	}

	err = c.target.SetSecret(secret)
	if err != nil {
		return err
	}

	return nil
}

// RemoveSecret is a no-op. Don't call it. Always returns an error.
func (c *Cacher) RemoveSecret(name string) error {
	return errors.New("not implemented")
}
