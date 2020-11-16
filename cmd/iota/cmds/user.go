
package cmds

import (
	"github.com/redhill42/iota/auth/userdb"
	"github.com/redhill42/iota/pkg/mflag"
)

func (cli *ServerCli) CmdUserAdd(args ...string) (err error) {
	cmd := cli.Subcmd("useradd", "USERNAME PASSWORD")
	cmd.Require(mflag.Min, 2)
	cmd.Require(mflag.Max, 2)
	cmd.ParseFlags(args, true)

	users, err := userdb.Open()
	if err != nil {
		return err
	}
	defer users.Close()

	user := &userdb.BasicUser{}
	user.Name = cmd.Arg(0)
	return users.Create(user, cmd.Arg(1))
}

func (cli *ServerCli) CmdUserDel(args ...string) error {
	cmd := cli.Subcmd("userdel", "USERNAME")
	cmd.Require(mflag.Exact, 1)
	cmd.ParseFlags(args, true)

	users, err := userdb.Open()
	if err != nil {
		return err
	}
	defer users.Close()
	return users.Remove(cmd.Arg(0))
}
