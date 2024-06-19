package builder

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

type InteropAPIBackend interface {
	SendGraphBundle(ctx context.Context, cb CrossBundle) error
}

type interopAPI struct {
	b InteropAPIBackend
}

func NewInteropAPI(b InteropAPIBackend) *interopAPI {
	return &interopAPI{b: b}
}

func GetInteropAPI(api *interopAPI) gethrpc.API {
	return gethrpc.API{
		Namespace: "eth",
		Service:   api,
	}
}

func (api *interopAPI) SendGraphBundle(ctx context.Context, cb CrossBundle) error {
	cross_bundle := CrossBundle{
		Messages: []Message{
			{
				ChainId: big.NewInt(21),
				Data:    common.Hex2Bytes("f8a8fd6d"),
				Target:  common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3"),
				Value:   big.NewInt(0),
			},
			{
				ChainId: big.NewInt(22),
				Data:    common.Hex2Bytes("f8a8fd6d"),
				Target:  common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3"),
				Value:   big.NewInt(0),
			},
			{
				ChainId: big.NewInt(23),
				Data:    common.Hex2Bytes("f8a8fd6d"),
				Target:  common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3"),
				Value:   big.NewInt(0),
			},
			{
				ChainId: big.NewInt(24),
				Data:    common.Hex2Bytes("f8a8fd6d"),
				Target:  common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3"),
				Value:   big.NewInt(0),
			},
		},
	}

	fmt.Println("Sending graph bundle")

	return api.b.SendGraphBundle(ctx, cross_bundle)
}
