package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/keeper"
	"github.com/zostay/dotfiles-go/internal/secrets"
)

var (
	getLocalOnly, getRemoteOnly, getMasterOnly bool
)

func init() {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a secret",
		Args:  cobra.ExactArgs(1),
		RunE:  RunGetSecret,
	}

	getCmd.Flags().BoolVarP(&getLocalOnly, "local-only", "l", false, "create secret only in local database")
	getCmd.Flags().BoolVarP(&getRemoteOnly, "remote-only", "r", false, "create secret only in remote database")
	getCmd.Flags().BoolVarP(&getMasterOnly, "master", "m", false, "set the secret in the system keyring")

	cmd.AddCommand(getCmd)
}

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
