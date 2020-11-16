package cmds

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/redhill42/iota/config"
	"github.com/redhill42/iota/pkg/gopass"
	"github.com/redhill42/iota/pkg/mflag"
	"github.com/redhill42/iota/pkg/rest"
)

func (cli *ClientCli) CmdLogin(args ...string) error {
	cmd := cli.Subcmd("login", "[USERNAME [PASSWORD]]")
	cmd.Require(mflag.Max, 2)
	cmd.ParseFlags(args, true)

	if err := cli.Connect(); err != nil {
		return err
	}

	var username, password string
	if cmd.NArg() > 0 {
		username = cmd.Arg(0)
	}
	if cmd.NArg() > 1 {
		password = cmd.Arg(1)
	}
	return cli.authenticate("Enter user credentials.", username, password)
}

func (cli *ClientCli) CmdLogout(args ...string) error {
	cmd := cli.Subcmd("logout", "")
	cmd.Require(mflag.Exact, 0)
	cmd.ParseFlags(args, true)

	if cli.host != "" {
		config.RemoveOption(cli.host, "token")
		config.Save()
	}
	return nil
}

func (c *ClientCli) authenticate(prompt, username, password string) (err error) {
	if username == "" || password == "" {
		fmt.Fprintln(c.stdout, prompt)
	}

	if username == "" {
		fmt.Fprintf(c.stdout, "Username: ")
		reader := bufio.NewReader(os.Stdin)
		username, err = reader.ReadString('\n')
		if err != nil {
			return err
		} else {
			username = strings.TrimSpace(username)
		}
	}

	if password == "" {
		fmt.Fprintf(c.stdout, "Password: ")
		pass, err := gopass.GetPasswdMasked()
		if err != nil {
			return err
		} else {
			password = string(pass)
		}
	}

	token, err := c.Authenticate(context.Background(), strings.ToLower(username), password)
	if err != nil {
		if se, ok := err.(rest.ServerError); ok && se.StatusCode() == http.StatusUnauthorized {
			err = errors.New("Login failed. Please enter valid user name and password.")
		}
		return err
	}

	c.SetToken(token)
	config.AddOption(c.host, "token", token)
	config.Save()
	return nil
}
