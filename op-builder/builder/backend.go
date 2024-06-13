package builder

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var CrossL2InboxAddress = common.HexToAddress("0x4200000000000000000000000000000000000022")
var PrivateKeyHex = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
var AddressHex = "f39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
var Address = common.HexToAddress(AddressHex)

type Backend struct {
	chains map[string]*ethclient.Client
	log    log.Logger
}

func NewBackend(clients map[string]*ethclient.Client, log log.Logger) *Backend {
	return &Backend{
		chains: clients,
		log:    log,
	}
}

type SendInteropBundleArgs struct {
	Txs         []hexutil.Bytes `json:"txs"`
	BlockNumber rpc.BlockNumber `json:"blockNumber"`
}

type SendInteropBundleResponse struct {
	BlockNumber uint64      `json:"blockNumber"`
	Timestamp   uint64      `json:"timestamp"`
	Logs        []types.Log `json:"logs"`
	Bundlehash  common.Hash `json:"bundlehash"`
}

func transactionToHex(tx *types.Transaction) (hexutil.Bytes, error) {
	// Encode the transaction to RLP
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// Convert the RLP encoded bytes to hexutil.Bytes
	hexBytes := hexutil.Bytes(txBytes)
	return hexBytes, nil
}

func sendInteropBundle(client *ethclient.Client, ctx context.Context, txBytes hexutil.Bytes) (*SendInteropBundleResponse, error) {
	var result *SendInteropBundleResponse
	args := SendInteropBundleArgs{
		Txs:         []hexutil.Bytes{txBytes},
		BlockNumber: rpc.PendingBlockNumber,
	}

	err := client.Client().CallContext(ctx, &result, "eth_sendInteropBundle", args)
	if err != nil {
		return nil, fmt.Errorf("failed to send interop bundle: %w", err)
	}

	return result, nil
}

func (b *Backend) newTxFromMessage(ctx context.Context, msg *Message) (*types.Transaction, error) {
	client, ok := b.chains[msg.ChainId.String()]
	if !ok {
		return nil, fmt.Errorf("chain %s not found in the dependency list", msg.ChainId)
	}

	privateKey, err := crypto.HexToECDSA(PrivateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key: %w", err)
	}

	nonce, err := client.PendingNonceAt(ctx, Address)

	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to get latest block header: %v", err)
	}
	baseFee := header.BaseFee
	maxPriorityFeePerGas := big.NewInt(2e9) // 2 gwei
	maxFeePerGas := new(big.Int).Add(baseFee, maxPriorityFeePerGas)
	gasLimit := uint64(200_000)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   msg.ChainId,
		Nonce:     nonce,
		GasTipCap: maxPriorityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gasLimit,
		To:        &msg.Target,
		Value:     msg.Value,
		Data:      msg.Data,
	})

	signer := types.NewLondonSigner(msg.ChainId)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	return signedTx, nil
}

func (b *Backend) SendGraphBundle(ctx context.Context, cb CrossBundle) error {
	var id ICrossL2InboxIdentifier

	fmt.Println("Sending graph bundle", cb)
	for i, msg := range cb.Messages {
		fmt.Println("Processing message", msg)
		client, ok := b.chains[msg.ChainId.String()]
		if !ok {
			return fmt.Errorf("chain %s not found in the dependency list", msg.ChainId)
		}

		if i == 0 {
			b.log.Info("Sending initiating message", "msg", msg)
			tx, err := b.newTxFromMessage(ctx, &msg)
			if err != nil {
				return fmt.Errorf("failed to create transaction: %w", err)
			}

			b.log.Info("Created tx from message", "tx", tx)
			txBytes, err := transactionToHex(tx)
			if err != nil {
				return fmt.Errorf("failed to convert transaction to hex: %w", err)
			}
			b.log.Info("Converted tx to hex", "tx", txBytes)

			resp, err := sendInteropBundle(client, ctx, txBytes)
			if err != nil {
				return fmt.Errorf("failed to send interop bundle: %w", err)
			}

			id.LogIndex = big.NewInt(0)
			id.Timestamp = big.NewInt(int64(resp.Timestamp))
			id.BlockNumber = big.NewInt(int64(resp.BlockNumber))
			id.ChainId = msg.ChainId
			id.Origin = Address
		} else {
			b.log.Info("Sending executing message", "msg", msg)
			contract, err := NewCrossL2Inbox(CrossL2InboxAddress, client)
			if err != nil {
				return fmt.Errorf("failed to create contract: %w", err)
			}
			privKey, err := crypto.HexToECDSA(PrivateKeyHex)

			if err != nil {
				return fmt.Errorf("failed to create private key: %w", err)
			}
			opts, err := bind.NewKeyedTransactorWithChainID(privKey, msg.ChainId)
			if err != nil {
				return fmt.Errorf("failed to create transactor: %w", err)
			}
			opts.NoSend = true

			tx, err := contract.ExecuteMessage(opts, id, msg.Target, msg.Data)
			if err != nil {
				return fmt.Errorf("failed to create tx: %w", err)
			}

			r, v, s := tx.RawSignatureValues()
			b.log.Info("Signature", r, v, s)

			txBytes, err := transactionToHex(tx)
			if err != nil {
				return fmt.Errorf("failed to convert transaction to hex: %w", err)
			}

			resp, err := sendInteropBundle(client, ctx, txBytes)
			if err != nil {
				return fmt.Errorf("failed to send interop bundle: %w", err)
			}
			id.LogIndex = big.NewInt(0)
			id.Timestamp = big.NewInt(int64(resp.Timestamp))
			id.BlockNumber = big.NewInt(int64(resp.BlockNumber))
			id.ChainId = msg.ChainId
			id.Origin = Address
		}
	}
	return nil
}
