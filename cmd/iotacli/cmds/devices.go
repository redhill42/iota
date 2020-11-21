package cmds

import (
	"context"
	"fmt"
	"os"

	"encoding/json"
	"github.com/redhill42/iota/pkg/mflag"
)

const devicesCmdUsage = `Usage: iotacli device [ID]

list devices or show device attributes (if an ID is provided).

Additional commands, type iotacli help COMMAND for more details:

  device:create      Create a new device
  device:update      Update a device's attributes
  device:remove      Permanently remove a device
  device:rpc         Make a remote procedure call on a device
`

func (cli *ClientCli) CmdDevice(args ...string) error {
	var help bool
	var err error

	cmd := cli.Subcmd("device", "[ID]")
	cmd.Require(mflag.Min, 0)
	cmd.Require(mflag.Max, 1)
	cmd.BoolVar(&help, []string{"-help"}, false, "Print usage")
	cmd.ParseFlags(args, false)

	if help {
		fmt.Fprintln(cli.stdout, devicesCmdUsage)
		os.Exit(0)
	}

	if err = cli.ConnectAndLogin(); err != nil {
		return err
	}

	if cmd.NArg() == 0 {
		if devices, err := cli.GetDevices(context.Background()); err == nil {
			for _, id := range devices {
				fmt.Fprintln(cli.stdout, id)
			}
		}
	} else {
		id := cmd.Arg(0)
		info := make(map[string]interface{})
		if err = cli.GetDevice(context.Background(), id, &info); err == nil {
			cli.writeJson(info)
		}
	}
	return err
}

func (cli *ClientCli) CmdDeviceCreate(args ...string) error {
	cmd := cli.Subcmd("device:create", "ID [ATTRIBUTES]")
	cmd.Require(mflag.Min, 1)
	cmd.Require(mflag.Max, 2)
	cmd.ParseFlags(args, true)

	id := cmd.Arg(0)
	attributes := make(map[string]interface{})

	if cmd.NArg() == 2 {
		if err := json.Unmarshal([]byte(cmd.Arg(1)), &attributes); err != nil {
			return err
		}
	}

	if err := cli.ConnectAndLogin(); err != nil {
		return err
	}

	attributes["id"] = id
	token, err := cli.CreateDevice(context.Background(), attributes)
	if err == nil {
		fmt.Fprintln(cli.stdout, token)
	}
	return err
}

func (cli *ClientCli) CmdDeviceUpdate(args ...string) error {
	cmd := cli.Subcmd("device:update", "ID ATTRIBUTES")
	cmd.Require(mflag.Exact, 2)
	cmd.ParseFlags(args, true)

	id := cmd.Arg(0)
	attributes := make(map[string]interface{})
	if err := json.Unmarshal([]byte(cmd.Arg(1)), &attributes); err != nil {
		return err
	}

	if err := cli.ConnectAndLogin(); err != nil {
		return err
	}
	return cli.UpdateDevice(context.Background(), id, attributes)
}

func (cli *ClientCli) CmdDeviceDelete(args ...string) error {
	var yes bool

	cmd := cli.Subcmd("device:delete", "ID")
	cmd.Require(mflag.Exact, 1)
	cmd.BoolVar(&yes, []string{"y"}, false, "Confirm 'yes' to remove the application")
	cmd.ParseFlags(args, true)

	if !yes && !cli.confirm("You will lost all your device data") {
		return nil
	}
	if err := cli.ConnectAndLogin(); err != nil {
		return err
	}
	return cli.DeleteDevice(context.Background(), cmd.Arg(0))
}

func (cli *ClientCli) CmdDeviceRPC(args ...string) error {
	var requestId string

	cmd := cli.Subcmd("device:rpc", "[OPTIONS] ID REQUEST")
	cmd.Require(mflag.Exact, 2)
	cmd.StringVar(&requestId, []string{"i"}, "", "Request identifier")
	cmd.ParseFlags(args, true)

	id := cmd.Arg(0)
	req := make(map[string]interface{})

	if err := cli.ConnectAndLogin(); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(cmd.Arg(1)), &req); err != nil {
		return err
	}
	return cli.RPC(context.Background(), id, requestId, req)
}
