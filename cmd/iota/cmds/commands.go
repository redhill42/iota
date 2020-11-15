package cmds

import (
	Cli "github.com/redhill42/iota/pkg/cli"
	flag "github.com/redhill42/iota/pkg/mflag"
)

// Command is the struct containing the command name and description
type Command struct {
	Name        string
	Description string
}

type ServerCli struct {
	*Cli.Cli
	handlers map[string]func(...string) error
}

// Commands lists the top level commands and their short usage
var CommandUsage = []Command{
	{"api-server", "Start the API server"},
	{"config", "Get or set a configuration value"},
}

var Commands = make(map[string]Command)

func init() {
	for _, cmd := range CommandUsage {
		Commands[cmd.Name] = cmd
	}
}

func Init() *ServerCli {
	cli := new(ServerCli)
	cli.Cli = Cli.New("iota", cli)
	cli.Description = "IOTA management tool"
	cli.handlers = map[string]func(...string) error{
		"api-server": cli.CmdAPIServer,
		"config":     cli.CmdConfig,
	}
	return cli
}

func (cli *ServerCli) Command(name string) func(...string) error {
	return cli.handlers[name]
}

func (cli *ServerCli) Subcmd(name string, synopses ...string) *flag.FlagSet {
	var description string
	if cmd, ok := Commands[name]; ok {
		description = cmd.Description
	}
	return cli.Cli.Subcmd(name, synopses, description, true)
}
