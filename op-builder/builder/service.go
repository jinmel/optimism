package builder

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-service/dial"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type BuilderService struct {
	Log     log.Logger
	Clients map[string]*ethclient.Client

	Version   string
	rpcServer *oprpc.Server

	stopped atomic.Bool
}

func NewBuilderService(ctx context.Context, version string, cfg *CLIConfig, log log.Logger) (*BuilderService, error) {
	var bs BuilderService
	if err := bs.initFromCLIConfig(ctx, version, cfg, log); err != nil {
		return nil, errors.Join(err, bs.Stop(ctx))
	}

	return &bs, nil
}

func (bs *BuilderService) initFromCLIConfig(ctx context.Context,
	version string, cfg *CLIConfig, log log.Logger) error {
	bs.Version = version
	bs.Log = log

	if err := bs.initRPCServer(cfg); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}

	if err := bs.initRPCClients(ctx, cfg); err != nil {
		return fmt.Errorf("failed to start eth RPC clients: %w", err)
	}

	return nil
}

func (bs *BuilderService) Start(ctx context.Context) error {
	bs.Log.Info("Starting builder")
	return nil
}

func (bs *BuilderService) Stop(ctx context.Context) error {
	bs.Log.Info("Stopping builder")
	var result error
	if bs.rpcServer != nil {
		if err := bs.rpcServer.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop RPC server: %w", err))
		}
	}

	if result == nil {
		bs.stopped.Store(true)
		bs.Log.Info("interop builder stopped")
	}

	return nil
}

func (bs *BuilderService) Stopped() bool {
	return bs.stopped.Load()
}

func (bs *BuilderService) initRPCServer(cfg *CLIConfig) error {
	server := oprpc.NewServer(
		cfg.RPC.ListenAddr,
		cfg.RPC.ListenPort,
		bs.Version,
		oprpc.WithLogger(bs.Log),
	)

	backend := NewBackend(bs.Clients)
	graphAPI := NewInteropAPI(backend)
	server.AddAPI(GetInteropAPI(graphAPI))
	bs.Log.Info("BundleGraph API Enabled")

	bs.Log.Info("Starting RPC server", "addr", cfg.RPC.ListenAddr, "port", cfg.RPC.ListenPort)
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}
	bs.rpcServer = server
	return nil
}

func (bs *BuilderService) initRPCClients(ctx context.Context, cfg *CLIConfig) error {
	for chainId, rpcUrl := range cfg.Chains {
		client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, bs.Log, rpcUrl)
		if err != nil {
			return fmt.Errorf("failed to dial RPC client for chain %s: %w", chainId, err)
		}
		bs.Clients[chainId] = client
	}
	return nil
}
