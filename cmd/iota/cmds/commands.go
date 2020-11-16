package cmds

import (
	"github.com/redhill42/iota/pkg/cli"
	"github.com/redhill42/iota/pkg/mflag"
)

// Command is the struct containing the command name and description
type Command struct {
	Name        string
	Description string
}

type ServerCli struct {
	*cli.Cli
	handlers map[string]func(...string) error
}

// Commands lists the top level commands and their short usage
var CommandUsage = []Command{
	{"api-server", "Start the API server"},
	{"config", "Get or set a configuration value"},
	{"useradd", "Add a user"},
	{"userdel", "Remove a user"},
}

var Commands = make(map[string]Command)

func init() {
	for _, cmd := range CommandUsage {
		Commands[cmd.Name] = cmd
	}
}

func Init() *ServerCli {
	c := new(ServerCli)
	c.Cli = cli.New("iota", c)
	c.Description = "IOTA management tool"
	c.handlers = map[string]func(...string) error{
		"api-server": c.CmdAPIServer,
		"config":     c.CmdConfig,
		"useradd":    c.CmdUserAdd,
		"userdel":    c.CmdUserDel,
	}
	return c
}

func (c *ServerCli) Command(name string) func(...string) error {
	return c.handlers[name]
}

func (c *ServerCli) Subcmd(name string, synopses ...string) *mflag.FlagSet {
	var description string
	if cmd, ok := Commands[name]; ok {
		description = cmd.Description
	}
	return c.Cli.Subcmd(name, synopses, description, true)
}
