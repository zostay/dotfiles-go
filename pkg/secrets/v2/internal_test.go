package secrets_test

import (
	"testing"

	"github.com/zostay/dotfiles-go/pkg/secrets/v2"
	"github.com/zostay/dotfiles-go/pkg/secrets/v2/keepertest"
)

func TestInternal(t *testing.T) {
	factory := func() (secrets.Keeper, error) {
		return secrets.NewInternal()
	}

	ts := keepertest.New(factory)
	ts.Run(t)
}
