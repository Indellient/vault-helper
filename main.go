//
// Copyright 2018 Indellient Inc. - All Rights Reserved
//
// vault-helper: A CLI tool to fetch secrets from Vault and emit them on STDOUT, or parse them (using template
//               placeholders) in a text file, rendering the secrets in the file directly.
//
package main

import (
	"context"
	"os"

	"github.com/Indellient/vault-helper/pkg/cli"
)

// These vars describe buildtime variables that are emitted when help or version info is printed.
var (
	// Stores the at-build-time version, like "1.1.99"
	BuildVersion string

	// Stores the at-build-time timestamp, like "2012-10-31 15:50:13.793654 +0000 UTC"
	BuildTimestamp string
)

func main() {
	// Start up our context var, which we pass down to other pkgs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Parse the cli arguments, and perform the action(s)
	cli.BuildVersion = BuildVersion
	cli.BuildTimestamp = BuildTimestamp
	cli.Run(ctx, os.Args)
}
