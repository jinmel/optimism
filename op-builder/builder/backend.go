package builder

import (
	"bytes"
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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

var CrossL2InboxAddress = common.HexToAddress("0x4200000000000000000000000000000000000022")
var PrivateKeyHex = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
var AddressHex = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
var Address = common.HexToAddress(AddressHex)

type Backend struct {
	chains map[string]*ethclient.Client
	log    log.Logger
}

func NewBackend(clients map[string]*ethclient.Client) *Backend {
	return &Backend{
		chains: clients,
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
	rlpBytes, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}

	// Convert the RLP encoded bytes to hexutil.Bytes
	hexBytes := hexutil.Bytes(rlpBytes)

	return hexBytes, nil
}

func sendInteropBundle(client *ethclient.Client, ctx context.Context, txs types.Transactions) (*SendInteropBundleResponse, error) {
	number, err := client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get block number: %w", err)
	}

	txBytes := make([]hexutil.Bytes, len(txs))
	for i, tx := range txs {
		hexTx, err := transactionToHex(tx)
		if err != nil {
			return nil, fmt.Errorf("failed to convert transaction to hex: %w", err)
		}

		txBytes[i] = hexTx
	}

	var result *SendInteropBundleResponse
	err = client.Client().CallContext(ctx, &result, "eth_sendInteropBundle", SendInteropBundleArgs{
		Txs:         txBytes,
		BlockNumber: rpc.BlockNumber(number),
	})

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

	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to get latest block header: %v", err)
	}
	baseFee := header.BaseFee
	maxPriorityFeePerGas := big.NewInt(2e9) // 2 gwei
	maxFeePerGas := new(big.Int).Add(baseFee, maxPriorityFeePerGas)
	gasLimit := uint64(200_000)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(1), // Mainnet
		Nonce:     nonce,
		GasTipCap: maxPriorityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gasLimit,
		To:        &msg.Target,
		Value:     msg.Value,
		Data:      msg.Data,
	})

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(msg.ChainId), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return signedTx, nil
}

func (b *Backend) SendGraphBundle(ctx context.Context, cb *CrossBundle) error {
	var id ICrossL2InboxIdentifier

	for i, msg := range cb.Messages {
		client, ok := b.chains[msg.ChainId.String()]
		if !ok {
			return fmt.Errorf("chain %s not found in the dependency list", msg.ChainId)
		}

		if i == 0 {
			tx, err := b.newTxFromMessage(ctx, msg)
			if err != nil {
				return fmt.Errorf("failed to create transaction: %w from message %s", err, msg)
			}

			resp, err := sendInteropBundle(client, ctx, types.Transactions{tx})
			if err != nil {
				return fmt.Errorf("failed to send interop bundle: %w", err)
			}

			for _, log := range resp.Logs {
				m := make([]byte, 0)
				for _, topic := range log.Topics {
					m = append(m, topic.Bytes()...)
				}
				m = append(m, log.Data...)

				if bytes.Equal(m, msg.Payload) {
					id.LogIndex = big.NewInt(int64(log.Index))
					break
				}
			}
			id.Timestamp = big.NewInt(int64(resp.Timestamp))
			id.BlockNumber = big.NewInt(int64(resp.BlockNumber))
			id.ChainId = msg.ChainId
			id.Origin = Address
		} else {
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
			tx, err := contract.ExecuteMessage(opts, id, msg.Target, msg.Data)
			if err != nil {
				return fmt.Errorf("failed to create tx: %w", err)
			}

			resp, err := sendInteropBundle(client, ctx, types.Transactions{tx})
			if err != nil {
				return fmt.Errorf("failed to send interop bundle: %w", err)
			}
			for _, log := range resp.Logs {
				m := make([]byte, 0)
				for _, topic := range log.Topics {
					m = append(m, topic.Bytes()...)
				}
				m = append(m, log.Data...)

				if bytes.Equal(m, msg.Payload) {
					id.LogIndex = big.NewInt(int64(log.Index))
					break
				}
			}
			id.Timestamp = big.NewInt(int64(resp.Timestamp))
			id.BlockNumber = big.NewInt(int64(resp.BlockNumber))
			id.ChainId = msg.ChainId
			id.Origin = Address
		}
	}
	return nil
}
