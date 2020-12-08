package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/keeper"
	"github.com/zostay/dotfiles-go/internal/secrets"
)

func RunGetSecret(cmd *cobra.Command, args []string) error {
	keeper.RequiresSecretKeeper()

	if getLocalOnly && getRemoteOnly || getLocalOnly && getMasterOnly || getRemoteOnly && getMasterOnly {
		return errors.New("Only one of these options may be specified: --local-only/-l, --remote-only/-r, --master/-m")
	}

	var k secrets.Keeper
	if getMasterOnly {
		k = secrets.Master
	} else {
		lt := secrets.NewLocumTenens()

		if !getLocalOnly && !getMasterOnly {
			kp, err := secrets.NewKeepass()
			if err != nil {
				panic(err)
			}

			lt.AddKeeper(kp)
		}

		if !getRemoteOnly && !getMasterOnly {
			lp, err := secrets.NewLastPass()
			if err != nil {
				panic(err)
			}

			lt.AddKeeper(lp)
		}

		k = lt
	}

	name := args[0]

	secret, err := k.GetSecret(name)
	if err != nil {
		panic(err)
	}
	fmt.Println(secret)

	return nil
}
