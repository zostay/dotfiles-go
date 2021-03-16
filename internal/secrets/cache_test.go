package secrets

import (
	"testing"
	"time"
)

func TestCacher(t *testing.T) {
	factory := func() (Keeper, error) {
		src, err := NewInternal()
		if err != nil {
			return nil, err
		}

		tgt, err := NewInternal()
		if err != nil {
			return nil, err
		}

		return NewCacher(src, tgt, 24*time.Hour), nil
	}

	SecretKeeperTestSuite(t, factory)
}
