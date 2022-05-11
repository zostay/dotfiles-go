package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/mitchellh/go-wordwrap"
	"github.com/nsf/termbox-go"
	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/keeper"
	"github.com/zostay/dotfiles-go/pkg/nasapod"
	"github.com/zostay/dotfiles-go/pkg/secrets"
)

var (
	cmd *cobra.Command
	c   *nasapod.Client

	download bool
	output   string
	which    string
	count    int

	width int
)

func init() {
	cmd = &cobra.Command{
		Use:   "nasapod",
		Short: "Work with NASA Picture of the Day",
	}

	cmd.PersistentFlags().BoolVarP(&download, "download", "d", false, "Download the file")
	cmd.PersistentFlags().StringVarP(&output, "output", "o", "", "Choose the file name for the download")
	cmd.PersistentFlags().StringVarP(&which, "which", "w", "auto", "Set to auto, sd, hd, or thumb to select the image to download")

	todayCmd := &cobra.Command{
		Use:   "today",
		Short: "Fetch today's NASA Picture of the Day",
		Args:  cobra.NoArgs,
		Run:   RunToday,
	}

	dateCmd := &cobra.Command{
		Use:   "date <date>",
		Short: "Fetch the NASA Picture of the Day from another date",
		Args:  cobra.ExactArgs(1),
		Run:   RunDate,
	}

	rangeCmd := &cobra.Command{
		Use:   "range <start> <end>",
		Short: "Fetch the NASA Pictures of the Day from a range of dates",
		Args:  cobra.ExactArgs(2),
		Run:   RunRange,
	}

	rangeCmd.Flags().IntVar(&count, "count", 0, "Set to a non-zero value to limit the number of images to fetch")

	randomCmd := &cobra.Command{
		Use:   "random",
		Short: "Fetch a NASA Picture of the Day at random",
		Args:  cobra.NoArgs,
		Run:   RunRandom,
	}

	randomCmd.Flags().IntVar(&count, "count", 0, "Set to a non-zero value to limit the number of images to fetch")

	cmd.AddCommand(todayCmd)
	cmd.AddCommand(dateCmd)
	cmd.AddCommand(rangeCmd)
	cmd.AddCommand(randomCmd)

	keeper.RequiresSecretKeeper()
	apiKey := secrets.MustGet(secrets.Secure, "NASA_API_KEY")
	c = nasapod.New(apiKey)

	if err := termbox.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error determining terminal size: %v\n", err)
		os.Exit(1)
	}

	width, _ = termbox.Size()
	termbox.Close()
}

func indent(s string, n int) string {
	s = wordwrap.WrapString(s, uint(width-24))
	return strings.Replace(s, "\n", "\n"+strings.Repeat(" ", n), -1)
}

func printField(title, v string) {
	fmt.Printf("%20s: %s\n", title, indent(v, 22))
}

func outputMetadata(m *nasapod.Metadata) {
	printField("Title", m.Title)
	printField("Date", m.Date.Format("2006-01-02"))
	printField("Explanation", m.Explanation)
	printField("Type", m.MediaType)
}

func downloadImage(m *nasapod.Metadata) {
	var (
		fetchCmd  func(m *nasapod.Metadata) (string, io.Reader, error)
		givenName string
		dlType    string
	)
	switch which {
	case "auto":
		if m.HdUrl != "" {
			fetchCmd = c.FetchHdImage
			givenName = path.Base(m.HdUrl)
			dlType = "HD"
		} else if m.Url != "" {
			fetchCmd = c.FetchImage
			givenName = path.Base(m.Url)
			dlType = "SD"
		} else {
			fetchCmd = c.FetchThumbnailImage
			givenName = path.Base(m.ThumbnailUrl)
			dlType = "thumbnail"
		}
	case "sd":
		fetchCmd = c.FetchImage
		givenName = path.Base(m.Url)
		dlType = "SD"
	case "hd":
		fetchCmd = c.FetchHdImage
		givenName = path.Base(m.HdUrl)
		dlType = "HD"
	case "thumb":
		fetchCmd = c.FetchThumbnailImage
		givenName = path.Base(m.ThumbnailUrl)
		dlType = "thumbnail"
	default:
		fmt.Fprintf(os.Stderr, "Incorrect --which setting. It must be one of: auto, sd, hd, or thumb, but got %q\n", which)
		os.Exit(1)
	}

	_, rd, err := fetchCmd(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed while downloading image: %v", err)
		os.Exit(1)
	}

	fileName := givenName
	if output != "" {
		fileName = output
	}

	f, err := os.Create(fileName)
	defer func() {
		err := f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error saving filename %q for download: %v", fileName, err)
			os.Exit(1)
		}
	}()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating filename %q for download: %v", fileName, err)
		os.Exit(1)
	}

	_, err = io.Copy(f, rd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to filename %q for download: %v", fileName, err)
		os.Exit(1)
	}

	title := fmt.Sprintf("Downloaded %s", dlType)
	printField(title, fileName)
}

func parseDate(arg string) time.Time {
	d, err := time.Parse("2006-01-02", arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse date %q. It must be in YYYY-MM-DD format.", arg)
		os.Exit(1)
	}
	return d
}

func handleOne(r *nasapod.Request) {
	m, err := c.Execute(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed while fulfilling your request: %v\n", err)
		os.Exit(1)
	}

	if len(m) == 0 {
		fmt.Fprintln(os.Stderr, "Did not find a picture of the day.")
		os.Exit(1)
	}

	if len(m) > 1 {
		fmt.Fprintf(os.Stderr, "Expected a single picture of the day, but got %d.\n", len(m))
		os.Exit(1)
	}

	outputMetadata(&m[0])
	if download {
		downloadImage(&m[0])
	}
}

func handleMany(r *nasapod.Request) {
	m, err := c.Execute(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed while fulfilling your request: %v\n", err)
		os.Exit(1)
	}

	if len(m) == 0 {
		fmt.Fprintln(os.Stderr, "Did not find a picture of the day.")
		os.Exit(1)
	}

	for i := range m {
		mm := &m[i]
		outputMetadata(mm)
		if download {
			downloadImage(mm)
		}
	}
}

func RunToday(cmd *cobra.Command, args []string) {
	r, err := nasapod.NewRequest(nasapod.WithThumbs())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to fulfill your request: %v\n", err)
		os.Exit(1)
	}

	handleOne(r)
}

func RunDate(cmd *cobra.Command, args []string) {
	d := parseDate(args[0])

	r, err := nasapod.NewRequest(
		nasapod.WithThumbs(),
		nasapod.WithDate(d),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to fulfill your request: %v\n", err)
		os.Exit(1)
	}

	handleOne(r)
}

func RunRange(cmd *cobra.Command, args []string) {
	sd := parseDate(args[0])
	ed := parseDate(args[1])

	opts := []nasapod.Option{
		nasapod.WithThumbs(),
		nasapod.WithDateRange(sd, ed),
	}

	if count != 0 {
		opts = append(opts, nasapod.WithCount(count))
	}

	r, err := nasapod.NewRequest(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to fulfill your request: %v\n", err)
		os.Exit(1)
	}

	handleMany(r)
}

func RunRandom(cmd *cobra.Command, args []string) {
	if count == 0 {
		count = 1
	}

	r, err := nasapod.NewRequest(
		nasapod.WithThumbs(),
		nasapod.WithCount(count),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to fulfill your request: %v\n", err)
		os.Exit(1)
	}

	handleMany(r)
}

func main() {
	err := cmd.Execute()
	cobra.CheckErr(err)
}
