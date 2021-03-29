package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zostay/dotfiles-go/internal/fssafe"
)

func TestLowSecurity(t *testing.T) {
	lss := make([]*fssafe.TestingLoaderSaver, 0)

	factory := func() (Keeper, error) {
		k, err := newKeepass("", "testing123", "Test")
		if !assert.NoError(t, err, "no error getting keepass") {
			return nil, err
		}

		ls := fssafe.NewTestingLoaderSaver()
		lss = append(lss, ls)
		k.LoaderSaver = ls

		return k, nil
	}

	SecretKeeperTestSuite(t, factory)

	for _, ls := range lss {
		for i, r := range ls.Readers {
			assert.Truef(t, r.Closed, "reader %d was closed", i)
		}
		for i, w := range ls.Writers {
			assert.True(t, w.Closed, "writer %d was closed", i)
		}
	}
}
