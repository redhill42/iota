package main

import (
	"fmt"
	"os"

	"github.com/redhill42/iota/cmd/iota/cmds"
	"github.com/redhill42/iota/config"
	"github.com/redhill42/iota/pkg/mflag"
	"github.com/sirupsen/logrus"

	// Load all user database plugins
	_ "github.com/redhill42/iota/auth/userdb/file"
	_ "github.com/redhill42/iota/auth/userdb/mongodb"
)

func main() {
	stdout := os.Stdout

	err := config.Initialize()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	mflag.Usage = func() {
		mflag.CommandLine.SetOutput(stdout)

		fmt.Fprint(stdout, "Usage: iota [OPTIONS] COMMAND [arg...]\n       iota [ --help ]\n")
		help := "\nCommands:\n\n"
		commands := cmds.CommandUsage
		for _, cmd := range commands {
			help += fmt.Sprintf("  %-12.12s%s\n", cmd.Name, cmd.Description)
		}
		fmt.Fprintf(stdout, "%s\n", help)

		fmt.Fprint(stdout, "Options:\n")
		mflag.PrintDefaults()
		fmt.Fprint(stdout, "\nRun 'iota COMMAND --help' for more information on a command.\n")
	}

	flgHelp := mflag.Bool([]string{"h", "-help"}, false, "Print usage")
	flgDebug := mflag.Bool([]string{"D", "-debug"}, false, "Debugging mode")

	mflag.Parse()

	if *flgHelp {
		// if global flag --help is present, regardless of what other options
		// and commands there are, just print the usage.
		mflag.Usage()
		return
	}

	if *flgDebug {
		config.Debug = true
		logrus.SetLevel(logrus.DebugLevel)
	}

	c := cmds.Init()
	if err := c.Run(mflag.Args()...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
