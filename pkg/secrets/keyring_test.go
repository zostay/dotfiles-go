package secrets

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func TestKeyring(t *testing.T) {
	// the author probably shouldn't have exposed this, but since we have it...
	keyring.MockInit()

	factory := func() (Keeper, error) {
		return NewKeyring("testing"), nil
	}

	SecretKeeperTestSuite(t, factory)
}
