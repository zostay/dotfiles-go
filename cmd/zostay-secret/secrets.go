package main

import (
	"errors"

	"github.com/zostay/dotfiles-go/internal/keeper"
	"github.com/zostay/dotfiles-go/internal/secrets"
)

func secretKeeper() (secrets.Keeper, error) {
	keeper.RequiresSecretKeeper()

	if localOnly && remoteOnly || localOnly && masterOnly || remoteOnly && masterOnly {
		return nil, errors.New("Only one of these options may be specified: --local-only/-l, --remote-only/-r, --master/-m")
	}

	var k secrets.Keeper
	if masterOnly {
		var err error
		k, err = secrets.Master()
		if err != nil {
			panic(err)
		}
	} else {
		lt := secrets.NewLocumTenens()

		if !localOnly && !masterOnly {
			var kp secrets.Keeper
			var err error
			if insecure {
				kp, err = secrets.InsecureLocal()
			} else {
				kp, err = secrets.SecureLocal()
			}

			if err != nil {
				panic(err)
			}

			lt.AddKeeper(kp)
		}

		if !remoteOnly && !masterOnly {
			var lp secrets.Keeper
			var err error
			if insecure {
				lp, err = secrets.InsecureMain()
			} else {
				lp, err = secrets.SecureMain()
			}

			if err != nil {
				panic(err)
			}

			lt.AddKeeper(lp)
		}

		k = lt
	}

	return k, nil
}
