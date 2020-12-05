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

var (
	getCmd *cobra.Command

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
