package builder

import (
	"context"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

type InteropAPIBackend interface {
	SendGraphBundle(ctx context.Context, cb GraphBundle) error
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

func (api *interopAPI) SendGraphBundle(ctx context.Context, cb GraphBundle) error {
	return api.b.SendGraphBundle(ctx, cb)
}
