package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/mail"
)

var (
	cmd        *cobra.Command
	allFolders bool
	mailDir    string
)

func init() {
	cmd = &cobra.Command{
		Use:   "label-mail",
		Short: "Sort my email in the local MailDir",
		Run:   RunLabelMail,
	}

	cmd.PersistentFlags().BoolVar(&allFolders, "a", false, "run against mail from all time")
	cmd.PersistentFlags().StringVar(&mailDir, "maildir", mail.DefaultMailDir, "the root directory for mail")
}

func RunLabelMail(cmd *cobra.Command, args []string) {
	rules, err := mail.LoadRules()
	if err != nil {
		panic(err)
	}

	filter, err := mail.NewFilter(mailDir, rules)
	if err != nil {
		panic(err)
	}

	if !allMail {
		filter.LimitFilterToRecent(2 * time.Hour)
	}

	actions, err := filter.LabelMessages()
	if err != nil {
		panic(err)
	}

	total := 0
	kw := 5
	cw := 1
	keys := make([]string, 0, len(actions))
	for key, count := range actions {
		total += count
		keys = append(keys, key)

		if len(key) > kw {
			kw = len(key)
		}

		countLen := len(strconv.Itoa(count))
		if countLen > cw {
			cw = countLen
		}
	}

	sort.Sort(sort.StringSlice(keys))

	kws := strconv.Itoa(kw)
	cws := strconv.Itoa(cw)

	if total > 0 {
		for _, key := range keys {
			fmt.Printf(" %-"+kws+"s : %"+cws+"d\n", key, actions[key])
		}

		fmt.Printf("%s %s", strings.Repeat("-", kw+2), strings.Repeat("-", cw+2))
		fmt.Printf(" %-"+kws+"s : %"+cws+"d\n", "Total", total)
	} else {
		fmt.Println("Nothing to do.")
	}
}

func main() {
	cmd.Execute()
}
