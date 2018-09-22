package cli

import (
	"context"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
)

var (
	app = kingpin.New(os.Args[0], fmt.Sprintf(`Description:
  A command-line vault secrets fetcher and parser.`))

	BuildTag       string
	BuildTimestamp string
)

func Run(ctx context.Context, args []string) {
	fmt.Println("Hello World", BuildTag, BuildTimestamp)
}
