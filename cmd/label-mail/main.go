package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/spf13/cobra"
	_ "github.com/zostay/go-addr/pkg/addr/encoding"
	_ "github.com/zostay/go-email/pkg/email/encoding"

	"github.com/zostay/dotfiles-go/internal/keeper"
	"github.com/zostay/dotfiles-go/internal/mail"
)

const VersionNumber = "2.1.0"

var (
	cmd            *cobra.Command
	allMail        bool
	mailDir        string
	rulesFile      string
	localRulesFile string
	dryRun         bool
	verbose        int
	folders        []string
	allowSending   bool
	cpuprofile     string
	vacuumFirst    bool
	vacuumOnly     bool
	version        bool
)

func init() {
	keeper.RequiresSecretKeeper()

	cmd = &cobra.Command{
		Use:   "label-mail",
		Short: "Sort my email in the local MailDir",
		Run:   RunLabelMail,
	}

	cmd.PersistentFlags().BoolVarP(&allMail, "all-mail", "a", false, "run against mail from all time")
	cmd.PersistentFlags().StringVar(&mailDir, "maildir", mail.DefaultMailDir, "the root directory for mail")
	cmd.PersistentFlags().StringVar(&rulesFile, "rules", mail.DefaultPrimaryRulesConfigPath(), "the primary rules file")
	cmd.PersistentFlags().StringVar(&localRulesFile, "local-rules", mail.DefaultLocalRulesConfigPath(), "the local rules file")
	cmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "d", false, "perform a dry run")
	cmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "enable debugging verbose mode")
	cmd.PersistentFlags().StringSliceVarP(&folders, "folder", "f", []string{}, "select folders to filter")
	cmd.PersistentFlags().BoolVarP(&allowSending, "allow-forwarding", "e", false, "allow email forwarding rules to run")
	cmd.PersistentFlags().StringVar(&cpuprofile, "cpuprofile", "", "write CPU profile to `file`")
	cmd.PersistentFlags().BoolVar(&vacuumFirst, "vacuum-first", false, "vacuum the Mail directory before filtering")
	cmd.PersistentFlags().BoolVar(&vacuumOnly, "vacuum-only", false, "vacuum the Mail directory without filtering")
	cmd.PersistentFlags().BoolVar(&version, "version", false, "show the version information for the program")
}

func RunLabelMail(cmd *cobra.Command, args []string) {
	if version {
		fmt.Printf("label-mail v%s\n", VersionNumber)
		return
	}

	if mailDir == "" {
		panic(errors.New("maildir did not work"))
	}

	filter, err := mail.NewFilter(mailDir, rulesFile, localRulesFile)
	if err != nil {
		panic(err)
	}

	filter.SetDebugLevel(verbose)
	filter.SetDryRun(dryRun)

	if !allMail {
		filter.LimitFilterToRecent(2 * time.Hour)
	}

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	if vacuumOnly {
		vacuumFirst = true
	}

	if vacuumFirst {
		err := filter.Vacuum()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	var actions mail.ActionsSummary
	if !vacuumOnly {
		actions, err = filter.LabelMessages(folders)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	fmt.Print(actions)
}

func main() {
	err := cmd.Execute()
	cobra.CheckErr(err)
}
