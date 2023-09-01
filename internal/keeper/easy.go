package keeper

import (
	"context"
	"fmt"

	"github.com/zostay/ghost/pkg/config"
	"github.com/zostay/ghost/pkg/keeper"
	"github.com/zostay/ghost/pkg/secrets"
)

func MustGetSecret(name string) secrets.Secret {
	sec, err := GetSecret(name)
	if err != nil {
		panic(err)
	}
	return sec
}

func GetSecret(name string) (secrets.Secret, error) {
	c := config.Instance()
	ctx := keeper.WithBuilder(context.Background(), c)
	kpr, err := keeper.Build(ctx, c.MasterKeeper)
	if err != nil {
		return nil, err
	}

	secs, err := kpr.GetSecretsByName(ctx, name)
	if err != nil {
		return nil, err
	}

	switch len(secs) {
	case 0:
		return nil, fmt.Errorf("no secret named %q found", name)
	case 1:
		return secs[0], nil
	}

	return nil, fmt.Errorf("more than one secret named %q found", name)
}
