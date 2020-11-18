package cmds

import (
	"context"
	"fmt"
	"github.com/redhill42/iota/pkg/mflag"
)

func (cli *ClientCli) CmdVersion(args ...string) error {
	cmd := cli.Subcmd("version", "")
	cmd.Require(mflag.Exact, 0)
	cmd.ParseFlags(args, true)

	if err := cli.Connect(); err != nil {
		return err
	}

	cv := cli.ClientVersion()
	sv, err := cli.ServerVersion(context.Background())
	if err != nil {
		return err
	}

	fmt.Fprintln(cli.stdout, "Client:")
	fmt.Fprintf(cli.stdout, " Version:     %s\n", cv.Version)
	fmt.Fprintf(cli.stdout, " Git commit:  %s\n", cv.GitCommit)
	fmt.Fprintf(cli.stdout, " Build time:  %s\n", cv.BuildTime)

	fmt.Fprintf(cli.stdout, "\nServer: %s\n", cli.host)
	fmt.Fprintf(cli.stdout, " Version:     %s\n", sv.Version)
	fmt.Fprintf(cli.stdout, " Git commit:  %s\n", sv.GitCommit)
	fmt.Fprintf(cli.stdout, " Build time:  %s\n", sv.BuildTime)
	fmt.Fprintf(cli.stdout, " OS/Arch:     %s/%s\n", sv.Os, sv.Arch)

	return nil
}
