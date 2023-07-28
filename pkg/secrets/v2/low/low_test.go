package low_test

import (
	"testing"

	"github.com/zostay/dotfiles-go/internal/fssafe"
	"github.com/zostay/dotfiles-go/pkg/secrets/v2"
	"github.com/zostay/dotfiles-go/pkg/secrets/v2/keepertest"
	"github.com/zostay/dotfiles-go/pkg/secrets/v2/low"
)

func TestLowSecurity(t *testing.T) {
	factory := func() (secrets.Keeper, error) {
		return low.NewLowSecurityCustom(fssafe.NewTestingLoaderSaver()), nil
	}

	ts := keepertest.New(factory)
	ts.Run(t)
}
