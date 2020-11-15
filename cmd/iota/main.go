package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"

	"github.com/redhill42/iota/cmd/iota/cmds"
	"github.com/redhill42/iota/config"
	flag "github.com/redhill42/iota/pkg/mflag"
)

func main() {
	stdout := os.Stdout

	err := config.Initialize()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	flag.Usage = func() {
		flag.CommandLine.SetOutput(stdout)

		fmt.Fprint(stdout, "Usage: iota [OPTIONS] COMMAND [arg...]\n       iota [ --help ]\n")
		help := "\nCommands:\n\n"
		commands := cmds.CommandUsage
		for _, cmd := range commands {
			help += fmt.Sprintf("  %-12.12s%s\n", cmd.Name, cmd.Description)
		}
		fmt.Fprintf(stdout, "%s\n", help)

		fmt.Fprint(stdout, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprint(stdout, "\nRun 'iota COMMAND --help' for more information on a command.\n")
	}

	flgHelp := flag.Bool([]string{"h", "-help"}, false, "Print usage")
	flgDebug := flag.Bool([]string{"D", "-debug"}, false, "Debugging mode")

	flag.Parse()

	if *flgHelp {
		// if global flag --help is present, regardless of what other options
		// and commands there are, just print the usage.
		flag.Usage()
		return
	}

	if *flgDebug {
		config.Debug = true
		logrus.SetLevel(logrus.DebugLevel)
	}

	c := cmds.Init()
	if err := c.Run(flag.Args()...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
