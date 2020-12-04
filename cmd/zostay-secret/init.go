package main

import (
	"github.com/spf13/cobra"
)

var (
	cmd *cobra.Command
)

func init() {
	cmd = &cobra.Command{
		Use:   "zostay-secret",
		Short: "Work with my secrets",
	}
}

var (
	setCmd *cobra.Command

	localOnly, remoteOnly, masterOnly bool
)

func init() {
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set a secret",
		Args:  cobra.ExactArgs(2),
		RunE:  RunSetSecret,
	}

	setCmd.Flags().BoolVarP(&localOnly, "local-only", "l", false, "create secret only in local database")
	setCmd.Flags().BoolVarP(&remoteOnly, "remote-only", "r", false, "create secret only in remote database")
	setCmd.Flags().BoolVarP(&remoteOnly, "master", "m", false, "set the secret in the system keyring")

	cmd.AddCommand(setCmd)
}

var (
	keeperCmd *cobra.Command
)

func init() {
	keeperCmd := &cobra.Command{
		Use:   "keeper",
		Short: "Startup the secret keeper server",
		Run:   RunSecretKeeper,
	}

	cmd.AddCommand(keeperCmd)
}
