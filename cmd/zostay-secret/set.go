package main

import (
	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/secrets"
)

func initSet() {
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set a secret",
		Args:  cobra.ExactArgs(2),
		RunE:  RunSetSecret,
	}

	cmd.AddCommand(setCmd)
}

func RunSetSecret(cmd *cobra.Command, args []string) error {
	k, err := secretKeeper()
	if err != nil {
		panic(err)
	}

	name := args[0]
	secret := args[1]

	if masterOnly {
		err := secrets.SetMasterPassword(name, secret)
		if err != nil {
			panic(err)
		}
		return nil
	}

	err = k.SetSecret(&secrets.Secret{
		Name:  name,
		Value: secret,
	})
	if err != nil {
		panic(err)
	}

	return nil
}
