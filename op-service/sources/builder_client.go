package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	builderDeneb "github.com/attestantio/go-builder-client/api/deneb"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

var (
	errHTTPErrorResponse = errors.New("HTTP error response")
)

const PathGetPayload = "/eth/v1/builder/payload"

type BuilderAPIConfig struct {
	Endpoint string
}

func BuilderAPIDefaultConfig() *BuilderAPIConfig {
	return &BuilderAPIConfig{
		Endpoint: "",
	}
}

type BuilderAPIClient struct {
	log        log.Logger
	config     *BuilderAPIConfig
	httpClient *client.BasicHTTPClient
}

func NewBuilderAPIClient(log log.Logger, config *BuilderAPIConfig) *BuilderAPIClient {
	httpClient := client.NewBasicHTTPClient(config.Endpoint, log)

	return &BuilderAPIClient{
		httpClient: httpClient,
		config:     config,
		log:        log,
	}
}

func (s *BuilderAPIClient) Enabled() bool {
	return s.config.Endpoint != ""
}

func (s *BuilderAPIClient) GetPayload(ctx context.Context, ref eth.L2BlockRef, log log.Logger) (*eth.ExecutionPayloadEnvelope, error) {
	responsePayload := new(builderDeneb.SubmitBlockRequest)
	url := fmt.Sprintf("%s/%d/%s", PathGetPayload, ref.Number+1, ref.Hash)
	log.Info("Fetching payload", "url", url)
	header := http.Header{"Accept": {"application/json"}}
	resp, err := s.httpClient.Get(ctx, url, nil, header)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	log.Info("Response", "status", resp.Status, "header", resp.Header, "statuscode", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, errHTTPErrorResponse
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Info("Payload fetched", "payload", string(bodyBytes))

	if err := json.Unmarshal(bodyBytes, responsePayload); err != nil {
		return nil, err
	}

	return submitBlockRequestToExecutionPayloadEnvelope(responsePayload), nil
}

func submitBlockRequestToExecutionPayloadEnvelope(request *builderDeneb.SubmitBlockRequest) *eth.ExecutionPayloadEnvelope {
	txs := make([]eth.Data, len(request.ExecutionPayload.Transactions))

	for i, tx := range request.ExecutionPayload.Transactions {
		txs[i] = eth.Data(tx)
	}

	withdrawals := make([]*types.Withdrawal, len(request.ExecutionPayload.Withdrawals))
	for i, withdrawal := range request.ExecutionPayload.Withdrawals {
		withdrawals[i] = &types.Withdrawal{
			Index:     uint64(withdrawal.Index),
			Validator: uint64(withdrawal.ValidatorIndex),
			Address:   common.BytesToAddress(withdrawal.Address[:]),
			Amount:    uint64(withdrawal.Amount),
		}
	}

	ws := types.Withdrawals(withdrawals)

	payload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:    common.BytesToHash(request.ExecutionPayload.ParentHash[:]),
			FeeRecipient:  common.BytesToAddress(request.ExecutionPayload.FeeRecipient[:]),
			StateRoot:     eth.Bytes32(request.ExecutionPayload.StateRoot),
			ReceiptsRoot:  eth.Bytes32(request.ExecutionPayload.ReceiptsRoot),
			LogsBloom:     eth.Bytes256(request.ExecutionPayload.LogsBloom),
			PrevRandao:    eth.Bytes32(request.ExecutionPayload.PrevRandao),
			BlockNumber:   eth.Uint64Quantity(request.ExecutionPayload.BlockNumber),
			GasLimit:      eth.Uint64Quantity(request.ExecutionPayload.GasLimit),
			GasUsed:       eth.Uint64Quantity(request.ExecutionPayload.GasUsed),
			Timestamp:     eth.Uint64Quantity(request.ExecutionPayload.Timestamp),
			ExtraData:     eth.BytesMax32(request.ExecutionPayload.ExtraData),
			BaseFeePerGas: eth.Uint256Quantity(*request.ExecutionPayload.BaseFeePerGas),
			BlockHash:     common.BytesToHash(request.ExecutionPayload.BlockHash[:]),
			Transactions:  txs,
			Withdrawals:   &ws,
			BlobGasUsed:   nil,
			ExcessBlobGas: nil,
		},
		ParentBeaconBlockRoot: nil, // OP-Stack ecotone upgrade related field. Not needed for PoC.
	}
	return payload
}
