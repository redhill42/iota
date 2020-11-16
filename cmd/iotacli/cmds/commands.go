package cmds

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/redhill42/iota/api"
	"github.com/redhill42/iota/api/client"
	"github.com/redhill42/iota/cmd/iotacli/cmds/ansi"
	"github.com/redhill42/iota/config"
	"github.com/redhill42/iota/pkg/cli"
	flag "github.com/redhill42/iota/pkg/mflag"
)

// Command is the struct containing the command name and description
type Command struct {
	Name        string
	Description string
}

type ClientCli struct {
	*cli.Cli
	host string
	*client.APIClient
	stdout, stderr io.Writer
	handlers       map[string]func(...string) error
}

// Commands lists the top level commands and their short usage
var CommandUsage = []Command{
	{"version", "Show the version information"},
}

var Commands = make(map[string]Command)

func init() {
	for _, cmd := range CommandUsage {
		Commands[cmd.Name] = cmd
	}
}

func Init(host string, stdout, stderr io.Writer) *ClientCli {
	c := new(ClientCli)
	c.Cli = cli.New("iotacli", c)
	c.Description = "Iota client interface"
	c.host = host
	c.stdout = stdout
	c.stderr = stderr
	c.handlers = map[string]func(...string) error{
		"version": c.CmdVersion,
	}
	return c
}

func (c *ClientCli) Command(name string) func(...string) error {
	return c.handlers[name]
}

func (c *ClientCli) Subcmd(name string, synopses ...string) *flag.FlagSet {
	var description string
	if cmd, ok := Commands[name]; ok {
		description = cmd.Description
	}
	return c.Cli.Subcmd(name, synopses, description, true)
}

func (c *ClientCli) Connect() (err error) {
	if c.APIClient != nil {
		return nil
	}

	if c.host == "" {
		if c.host = config.Get("host"); c.host == "" {
			return errors.New("No remote host specified, please run iotacli with -H option")
		}
	}

	headers := map[string]string{
		"Accept": "application/json",
	}

	c.APIClient, err = client.NewAPIClient(c.host+"/api", api.APIVersion, nil, headers)
	return err
}

func (cli *ClientCli) confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprintf(cli.stdout, ansi.Danger("WARNING")+": "+prompt+", continue (yes/no)? ")
		answer, err := reader.ReadString('\n')
		if err == io.EOF {
			return false
		}
		if err != nil {
			return false
		}
		answer = strings.TrimSpace(answer)
		if answer == "no" || answer == "" {
			return false
		}
		if answer == "yes" {
			return true
		}
		fmt.Fprintln(cli.stdout, "Please answer yes or no.")
	}
}
