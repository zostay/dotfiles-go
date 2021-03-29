package secrets

import (
	"testing"
)

func TestInternal(t *testing.T) {
	factory := func() (Keeper, error) {
		return NewInternal()
	}

	SecretKeeperTestSuite(t, factory)
}
