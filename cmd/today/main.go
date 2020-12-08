package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bbrks/wrap"
	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/esv"
	"github.com/zostay/dotfiles-go/internal/keeper"
)

var (
	cmd *cobra.Command
)

func init() {
	cmd = &cobra.Command{
		Use:   "today",
		Short: "Read some scripture today",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show a specified scripture",
		Args:  cobra.MinimumNArgs(1),
		Run:   RunTodayShow,
	}

	randomCmd := &cobra.Command{
		Use:   "random",
		Short: "Pick a scripture to read at random",
		Args:  cobra.ExactArgs(0),
		Run:   RunTodayRandom,
	}

	cmd.AddCommand(randomCmd)
	cmd.AddCommand(showCmd)
}

func RunTodayRandom(cmd *cobra.Command, args []string) {
	keeper.RequiresSecretKeeper()

	rand.Seed(time.Now().UTC().UnixNano())
	v, err := esv.RandomVerse()
	if err != nil {
		panic(err)
	}
	fmt.Println(wrap.Wrap(v, 70))
}

func RunTodayShow(cmd *cobra.Command, args []string) {
	keeper.RequiresSecretKeeper()

	ref := strings.Join(args, " ")
	v, err := esv.GetVerse(ref)
	if err != nil {
		panic(err)
	}
	fmt.Println(wrap.Wrap(v, 70))
}

func main() {
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
