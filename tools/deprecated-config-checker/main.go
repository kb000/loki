package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/grafana/loki/tools/deprecated-config-checker/checker"
)

func main() {
	var cfg checker.CheckerConfig

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	cfg.RegisterFlags(fs)
	if err := fs.Parse(os.Args[1:]); err != nil {
		panic(err)
	}

	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	c, err := checker.NewChecker(cfg)
	if err != nil {
		panic(err)
	}

	deprecates := c.CheckDeprecated()
	if len(deprecates) > 0 {
		fmt.Printf("-- Deprecated configs --\n\n")
		for _, d := range deprecates {
			fmt.Printf("[*] %s\n\n", d)
		}
	}

	deletes := c.CheckDeleted()
	if len(deletes) > 0 {
		fmt.Printf("-- Deleted configs --\n\n")
		for _, d := range deletes {
			fmt.Printf("[-] %s\n\n", d)
		}
	}

	// TODO(salvacorts):
	// - If there is no deprecations and no deleted configs in the input file, validate the config.
	// - Flag to print all deprecated configs using the doc-generator library.
}
