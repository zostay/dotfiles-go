package secrets

import (
	"context"
	"testing"

	"github.com/ansd/lastpass-go"
)

type testLastPass struct {
	accounts []*lastpass.Account
}

func newTestLastPass() *testLastPass {
	return &testLastPass{
		accounts: make([]*lastpass.Account, 0),
	}
}

func (lp *testLastPass) Accounts(_ context.Context) ([]*lastpass.Account, error) {
	return lp.accounts, nil
}

func (lp *testLastPass) Update(_ context.Context, want *lastpass.Account) error {
	for i, a := range lp.accounts {
		if a.Name == want.Name {
			lp.accounts[i] = want
		}
	}
	return nil
}

func (lp *testLastPass) Add(_ context.Context, want *lastpass.Account) error {
	lp.accounts = append(lp.accounts, want)
	return nil
}

func TestLastPass(t *testing.T) {
	factory := func() (Keeper, error) {
		return &LastPass{
			lp:    newTestLastPass(),
			cat:   "Test",
			limit: true,
		}, nil
	}

	SecretKeeperTestSuite(t, factory)
}
