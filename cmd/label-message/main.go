package main

import (
	"errors"
	"fmt"
	"github.com/kr/pretty"
	"os"

	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/mail"
)

var (
	cmd            *cobra.Command
	mailDir        string
	rulesFile      string
	localRulesFile string
	verbose        int
	dryRun         bool
	allowSending   bool
)

func init() {
	cmd = &cobra.Command{
		Use:   "label-message <folder> <filename>",
		Short: "Sort a single email message",
		Run:   RunLabelMessage,
		Args:  cobra.ExactArgs(2),
	}

	cmd.PersistentFlags().StringVar(&mailDir, "maildir", mail.DefaultMailDir, "the root directory for mail")
	cmd.PersistentFlags().StringVar(&rulesFile, "rules", mail.DefaultPrimaryRulesConfigPath(), "the primary rules file")
	cmd.PersistentFlags().StringVar(&localRulesFile, "local-rules", mail.DefaultLocalRulesConfigPath(), "the local rules file")
	cmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "d", false, "perform a dry run")
	cmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "enable debugging verbose mode")
	cmd.PersistentFlags().BoolVarP(&allowSending, "allow-forwarding", "e", false, "allow email forwarding rules to run")
}

func RunLabelMessage(cmd *cobra.Command, args []string) {
	if mailDir == "" {
		panic(errors.New("maildir did not work"))
	}

	filter, err := mail.NewFilter(mailDir, rulesFile, localRulesFile)
	if err != nil {
		panic(err)
	}

	filter.SetDryRun(dryRun)
	filter.SetDebugLevel(verbose)

	if verbose > 3 {
		pretty.Print(filter.AllRules())
	}

	folder := args[0]
	fn := args[1]

	actions, err := filter.LabelMessage(folder, fn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	fmt.Print(actions)
}

func main() {
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
