package cmds

import (
	"context"
	"fmt"
	"github.com/redhill42/iota/api"
	"github.com/redhill42/iota/pkg/mflag"
)

func (cli *ClientCli) CmdVersion(args ...string) error {
	cmd := cli.Subcmd("version", "")
	cmd.Require(mflag.Exact, 0)
	cmd.ParseFlags(args, true)

	if err := cli.Connect(); err != nil {
		return err
	}

	v, err := cli.ServerVersion(context.Background())
	if err != nil {
		return err
	}

	fmt.Fprintln(cli.stdout, "Client:")
	fmt.Fprintf(cli.stdout, " Version:     %s\n", api.Version)
	fmt.Fprintf(cli.stdout, " Git commit:  %s\n", api.GitCommit)
	fmt.Fprintf(cli.stdout, " Build time:  %s\n", api.BuildTime)

	fmt.Fprintf(cli.stdout, "\nServer: %s\n", cli.host)
	fmt.Fprintf(cli.stdout, " Version:     %s\n", v.Version)
	fmt.Fprintf(cli.stdout, " Git commit:  %s\n", v.GitCommit)
	fmt.Fprintf(cli.stdout, " Build time:  %s\n", v.BuildTime)
	fmt.Fprintf(cli.stdout, " OS/Arch:     %s/%s\n", v.Os, v.Arch)

	return nil
}
