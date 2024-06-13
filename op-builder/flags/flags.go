package flags

import (
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/urfave/cli/v2"
)

const EnvVarPrefix = "OP_BUILDER"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	L2EthRpcListFlag = &cli.StringSliceFlag{
		Name:    "l2-eth-rpc",
		Usage:   "ChainID=rpc_url mapping for talking to multiple L2 Rpcs with different chain ids",
		EnvVars: prefixEnvVars("L2_ETH_RPC"),
	}
)

func init() {
	Flags = []cli.Flag{
		L2EthRpcListFlag,
	}

	Flags = append(Flags, oprpc.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, oplog.CLIFlags(EnvVarPrefix)...)
}

var Flags []cli.Flag
