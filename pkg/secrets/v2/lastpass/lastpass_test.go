package lastpass_test

import (
	"context"
	"testing"

	lpass "github.com/ansd/lastpass-go"
	"github.com/zostay/go-std/slices"

	"github.com/zostay/dotfiles-go/pkg/secrets/v2"
	"github.com/zostay/dotfiles-go/pkg/secrets/v2/keepertest"
	"github.com/zostay/dotfiles-go/pkg/secrets/v2/lastpass"
)

type testLastPass struct {
	accounts []*lpass.Account
}

func newTestLastPass() *testLastPass {
	return &testLastPass{
		accounts: make([]*lpass.Account, 0),
	}
}

func (lp *testLastPass) Accounts(_ context.Context) ([]*lpass.Account, error) {
	return lp.accounts, nil
}

func (lp *testLastPass) Update(_ context.Context, want *lpass.Account) error {
	for i, a := range lp.accounts {
		if a.Name == want.Name {
			lp.accounts[i] = want
		}
	}
	return nil
}

func (lp *testLastPass) Add(_ context.Context, want *lpass.Account) error {
	lp.accounts = append(lp.accounts, want)
	return nil
}

func (lp *testLastPass) Delete(_ context.Context, want *lpass.Account) error {
	for i, a := range lp.accounts {
		if a.ID == want.ID {
			lp.accounts = slices.Delete(lp.accounts, i)
			return nil
		}
	}
	return nil
}

func TestLastPass(t *testing.T) {
	factory := func() (secrets.Keeper, error) {
		return lastpass.NewLastPassWithClient(newTestLastPass())
	}

	ts := keepertest.New(factory)
	ts.Run(t)
}
