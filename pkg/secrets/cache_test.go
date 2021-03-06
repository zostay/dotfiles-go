package secrets

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCacheDownstream(t *testing.T) {
	src, err := NewInternal()
	require.NoError(t, err, "no error creating internal")

	tgt, err := NewInternal()
	require.NoError(t, err, "no error creating another internal")

	cch := NewCacher(src, tgt, 24*time.Hour)

	err = src.SetSecret(
		&Secret{
			Name:  "upstream",
			Value: "secret",
		},
	)
	require.NoError(t, err, "no error setting on source")

	s, err := cch.GetSecret("upstream")
	require.NoError(t, err, "no error getting on cacher")

	assert.Equal(t, "upstream", s.Name, "got upstream secret name")
	assert.Equal(t, "secret", s.Value, "got upstream secret value")

	s2, err := tgt.GetSecret("upstream")
	require.NoError(t, err, "no error getting on target")

	assert.Equal(t, "upstream", s2.Name, "got upstream secret name copied to target")
	assert.Equal(t, "secret", s2.Value, "got upstream secret value copied to target")
}

type getdeathkeeper struct{}

func (getdeathkeeper) GetSecret(name string) (*Secret, error) {
	return nil, errors.New("death")
}
func (getdeathkeeper) SetSecret(secret *Secret) error { return nil }
func (getdeathkeeper) RemoveSecret(name string) error { return nil }

func TestCacheDependence(t *testing.T) {
	src := getdeathkeeper{}

	tgt, err := NewInternal()
	require.NoError(t, err, "no error creating another internal")

	cch := NewCacher(src, tgt, 24*time.Hour)

	err = tgt.SetSecret(
		&Secret{
			Name:  "downstream",
			Value: "terces",
		},
	)
	require.NoError(t, err, "no error setting on target")

	s, err := cch.GetSecret("downstream")

	// This proves that the secret was retrieved from the target without
	// touching the source.
	require.NoError(t, err, "no error getting on cacher")

	assert.Equal(t, "downstream", s.Name, "got downstream secret name")
	assert.Equal(t, "terces", s.Value, "got downstream secret value")
}
