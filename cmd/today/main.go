package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/esv"
)

var (
	cmd *cobra.Command
)

func init() {
	cmd = &cobra.Command{
		Use:   "today",
		Short: "Read some scripture today",
		Run:   RunToday,
	}
}

func RunToday(cmd *cobra.Command, args []string) {
	rand.Seed(time.Now().UTC().UnixNano())
	//query := strings.Join(args, " ")
	v, err := esv.RandomVerse()
	if err != nil {
		panic(err)
	}
	fmt.Println(v)
}

func main() {
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
