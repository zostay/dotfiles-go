package main

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/keeper"
	"github.com/zostay/dotfiles-go/internal/secrets"
)

var (
	setLocalOnly, setRemoteOnly, setMasterOnly bool
)

func init() {
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set a secret",
		Args:  cobra.ExactArgs(2),
		RunE:  RunSetSecret,
	}

	setCmd.Flags().BoolVarP(&setLocalOnly, "local-only", "l", false, "create secret only in local database")
	setCmd.Flags().BoolVarP(&setRemoteOnly, "remote-only", "r", false, "create secret only in remote database")
	setCmd.Flags().BoolVarP(&setMasterOnly, "master", "m", false, "set the secret in the system keyring")

	cmd.AddCommand(setCmd)
}

func RunSetSecret(cmd *cobra.Command, args []string) error {
	keeper.RequiresSecretKeeper()

	if setLocalOnly && setRemoteOnly || setLocalOnly && setMasterOnly || setRemoteOnly && setMasterOnly {
		return errors.New("Only one of these options may be specified: --local-only/-l, --remote-only/-r, --master/-m")
	}

	ks := make([]secrets.Keeper, 0, 2)
	if !setLocalOnly {
		lp, err := secrets.SecureMain()
		if err != nil {
			panic(err)
		}

		ks = append(ks, lp)
	}

	if !setRemoteOnly {
		kp, err := secrets.SecureLocal()
		if err != nil {
			panic(err)
		}

		ks = append(ks, kp)
	}

	name := args[0]
	secret := args[1]

	if setMasterOnly {
		err := secrets.SetMasterPassword(name, secret)
		if err != nil {
			panic(err)
		}
		return nil
	}

	for _, k := range ks {
		err := k.SetSecret(&secrets.Secret{
			Name:  name,
			Value: secret,
		})
		if err != nil {
			panic(err)
		}
	}

	return nil
}
