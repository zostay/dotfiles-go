package main

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/secrets"
)

func RunSetSecret(cmd *cobra.Command, args []string) error {
	RequiresSecretKeeper()

	if localOnly && remoteOnly || localOnly && masterOnly || remoteOnly && masterOnly {
		return errors.New("Only one of these options may be specified: --local-only/-l, --remote-only/-r, --master/-m")
	}

	ks := make([]secrets.Keeper, 0, 2)
	if !localOnly {
		lp, err := secrets.NewLastPass()
		if err != nil {
			panic(err)
		}

		ks = append(ks, lp)
	}

	if !remoteOnly {
		kp, err := secrets.NewKeepass()
		if err != nil {
			panic(err)
		}

		ks = append(ks, kp)
	}

	name := args[0]
	secret := args[1]

	for _, k := range ks {
		err := k.SetSecret(name, secret)
		if err != nil {
			panic(err)
		}
	}

	return nil
}
