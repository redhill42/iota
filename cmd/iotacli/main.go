package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/redhill42/iota/cmd/iotacli/cmds"
	"github.com/redhill42/iota/config"
	"github.com/redhill42/iota/pkg/colorable"
	flag "github.com/redhill42/iota/pkg/mflag"
	"github.com/redhill42/iota/pkg/rest"
)

func main() {
	stdout := colorable.NewColorableStdout()
	stderr := colorable.NewColorableStderr()

	err := config.InitializeClient()
	if err != nil {
		fmt.Fprintln(stderr, err)
		os.Exit(1)
	}

	flag.Usage = func() {
		flag.CommandLine.SetOutput(stdout)

		fmt.Fprint(stdout, "Usage: iotacli [OPTIONS] COMMAND [arg...]\n       iotacli [ --help ]\n")

		help := "\nCommands:\n\n"
		commands := cmds.CommandUsage
		for _, cmd := range commands {
			if !strings.ContainsRune(cmd.Name, ':') {
				help += fmt.Sprintf("  %-12.12s%s\n", cmd.Name, cmd.Description)
			}
		}
		fmt.Fprintf(stdout, "%s\n", help)

		fmt.Fprint(stdout, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprint(stdout, "\nRun 'iotacli COMMAND --help' for more information on a command.\n")
	}

	flgHelp := flag.Bool([]string{"h", "-help"}, false, "Print usage")
	flgDebug := flag.Bool([]string{"D", "-debug"}, false, "Debugging mode")
	flgHost := flag.String([]string{"H", "-host"}, "", "Connect to remote host")

	flag.Parse()

	if *flgHelp {
		// if global flag --help is present, regardless of what other options
		// and commands there are, just print the usage
		flag.Usage()
		return
	}

	if *flgDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	var host string
	if *flgHost != "" {
		if host, err := parseHost(*flgHost); err != nil {
			fmt.Fprintln(stderr, err)
			os.Exit(1)
		} else {
			config.Set("host", host)
			config.Save()
		}
	}

	c := cmds.Init(host, stdout, stderr)
	if err := c.Run(flag.Args()...); err != nil {
		if se, ok := err.(rest.ServerError); ok && se.StatusCode() == http.StatusUnauthorized {
			fmt.Fprintln(stderr, "Your access token has been expired, please login again.")
		} else {
			fmt.Fprintln(stderr, err)
		}
		os.Exit(1)
	}
}

func parseHost(host string) (string, error) {
	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err != nil {
			return "", err
		} else {
			u.Path = ""
			return u.String(), nil
		}
	} else {
		return "http://" + host, nil
	}
}
