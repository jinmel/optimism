package builder

import (
	"strings"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

type CLIConfig struct {
	Chains map[string]string

	RPC       oprpc.CLIConfig
	LogConfig oplog.CLIConfig
}

func parseL2Rpc(ctx *cli.Context) map[string]string {
	opts := ctx.StringSlice(L2EthRpcListFlag.Name)

	var l2Rpc map[string]string
	for _, opt := range opts {
		parts := strings.Split(opt, "=")
		if len(parts) != 2 {
			log.Crit("Invalid L2 RPC option", "option", opt)
		}
		chainID := parts[0]
		rpcURL := parts[1]

		l2Rpc[chainID] = rpcURL
	}
	return l2Rpc
}

func NewConfig(ctx *cli.Context) *CLIConfig {
	return &CLIConfig{
		Chains: parseL2Rpc(ctx),

		RPC:       oprpc.ReadCLIConfig(ctx),
		LogConfig: oplog.ReadCLIConfig(ctx),
	}
}
