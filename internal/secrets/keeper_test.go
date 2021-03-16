package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type KeeperFactory func() (Keeper, error)

func SecretKeeperTestSuite(t *testing.T, f KeeperFactory) {
	SecretKeeperGetMissingTest(t, f)
	SecretKeeperSetAndGet(t, f)
}

func SecretKeeperGetMissingTest(t *testing.T, f KeeperFactory) {
	k, err := f()
	if !assert.NoError(t, err, "factory returns keeper") {
		return
	}

	s, err := k.GetSecret("missing")
	if !assert.Error(t, err, "missing secret returns error") {
		return
	}

	assert.Equal(t, ErrNotFound, err, "missing secret returns ErrNotFound")
	assert.Nil(t, s, "missing secret is nil")
}

func SecretKeeperSetAndGet(t *testing.T, f KeeperFactory) {
	k, err := f()
	if !assert.NoError(t, err, "factory returns keeper") {
		return
	}

	// create
	err = k.SetSecret(
		&Secret{
			Name:  "set1",
			Value: "secret1",
		},
	)

	if !assert.NoError(t, err, "setting doesn't error") {
		return
	}

	got, err := k.GetSecret("set1")
	if !assert.NoError(t, err, "getting doesn't error") {
		return
	}

	if !assert.NotNil(t, got, "got something") {
		return
	}

	assert.Equal(t, "set1", got.Name, "got secret name set1")
	assert.Equal(t, "secret1", got.Value, "got secret value secret1")

	// update
	err = k.SetSecret(
		&Secret{
			Name:  "set1",
			Value: "secret2",
		},
	)

	if !assert.NoError(t, err, "setting again doesn't error") {
		return
	}

	got, err = k.GetSecret("set1")
	if !assert.NoError(t, err, "getting again doesn't error") {
		return
	}

	if !assert.NotNil(t, got, "got something again") {
		return
	}

	assert.Equal(t, "set1", got.Name, "got secret name still set1")
	assert.Equal(t, "secret2", got.Value, "but got secret value changed to secret2")
}
