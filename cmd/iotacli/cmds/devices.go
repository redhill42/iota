package cmds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/redhill42/iota/pkg/mflag"
)

const devicesCmdUsage = `Usage: iotacli device [ID]

list devices or show device attributes (if an ID is provided).

Additional commands, type iotacli help COMMAND for more details:

  device:create      Create a new device
  device:update      Update a device's attributes
  device:remove      Permanently remove a device
  device:rpc         Make a remote procedure call on a device
  device:claims      Show current device claims
  device:approve     Approve a device claim
  device:reject      Reject a device claim
`

func (cli *ClientCli) CmdDevice(args ...string) error {
	var help bool
	var keys string
	var err error

	cmd := cli.Subcmd("device", "[ID]")
	cmd.Require(mflag.Min, 0)
	cmd.Require(mflag.Max, 1)
	cmd.BoolVar(&help, []string{"-help"}, false, "Print usage")
	cmd.StringVar(&keys, []string{"k", "-keys"}, "", "Show values for given keys")
	cmd.ParseFlags(args, false)

	if help {
		fmt.Fprintln(cli.stdout, devicesCmdUsage)
		os.Exit(0)
	}

	if err = cli.ConnectAndLogin(); err != nil {
		return err
	}

	if cmd.NArg() == 0 {
		devices := make([]map[string]interface{}, 0)
		if err = cli.GetDevices(context.Background(), keys, &devices); err == nil {
			cli.writeJson(devices)
		}
	} else {
		id := cmd.Arg(0)
		info := make(map[string]interface{})
		if err = cli.GetDevice(context.Background(), id, keys, &info); err == nil {
			if len(info) == 1 {
				for _, v := range info {
					fmt.Fprintln(cli.stdout, v)
				}
			} else {
				cli.writeJson(info)
			}
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

	cmd := cli.Subcmd("device:rpc", "[OPTIONS] ID METHOD [PARAMETER=VALUE...]")
	cmd.Require(mflag.Min, 2)
	cmd.StringVar(&requestId, []string{"i"}, "0", "Request identifier")
	cmd.ParseFlags(args, true)

	params := make(map[string]interface{})
	for i := 2; i < cmd.NArg(); i++ {
		p := cmd.Arg(i)
		s := strings.IndexRune(p, '=')
		if s == -1 {
			return errors.New("missing '=' in method parameter")
		}
		params[p[0:s]] = convert(p[s+1:])
	}

	id := cmd.Arg(0)
	req := map[string]interface{}{
		"id":     0,
		"method": cmd.Arg(1),
		"param":  params,
	}

	if err := cli.ConnectAndLogin(); err != nil {
		return err
	}
	return cli.RPC(context.Background(), id, requestId, req)
}

func convert(value string) interface{} {
	if len(value) >= 2 && strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return value[1 : len(value)-1]
	}
	if len(value) >= 2 && strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return value[1 : len(value)-1]
	}
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return value
}

func (cli *ClientCli) CmdDeviceClaims(args ...string) error {
	cmd := cli.Subcmd("device:claims", "")
	cmd.Require(mflag.Exact, 0)
	cmd.ParseFlags(args, false)

	if err := cli.ConnectAndLogin(); err != nil {
		return err
	}

	claims, err := cli.GetClaims(context.Background())
	if err == nil {
		cli.writeJson(claims)
	}
	return err
}

func (cli *ClientCli) CmdDeviceApprove(args ...string) error {
	cmd := cli.Subcmd("device:approve", "CLAIM-ID [UPDATES]")
	cmd.Require(mflag.Min, 1)
	cmd.Require(mflag.Max, 2)
	cmd.ParseFlags(args, false)

	claimId := cmd.Arg(0)
	updates := make(map[string]interface{})
	if cmd.NArg() == 2 {
		if err := json.Unmarshal([]byte(cmd.Arg(1)), &updates); err != nil {
			return err
		}
	}

	if err := cli.ConnectAndLogin(); err != nil {
		return err
	}

	token, err := cli.ApproveDevice(context.Background(), claimId, updates)
	if err == nil {
		fmt.Fprintln(cli.stdout, token)
	}
	return err
}

func (cli *ClientCli) CmdDeviceReject(args ...string) error {
	cmd := cli.Subcmd("device:reject", "CLAIM-ID")
	cmd.Require(mflag.Exact, 1)
	cmd.ParseFlags(args, false)

	if err := cli.ConnectAndLogin(); err != nil {
		return err
	}
	return cli.RejectDevice(context.Background(), cmd.Arg(0))
}
