package rpc

import (
	"context"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

type GraphAPIBackend interface {
	SendGraph(ctx context.Context) error
}

type graphAPI struct {
	b GraphAPIBackend
}

func NewGraphAPI() *graphAPI {
	return &graphAPI{}
}

func GetGraphAPI(api *graphAPI) gethrpc.API {
	return gethrpc.API{
		Namespace: "eth",
		Service:   api,
	}
}

func (api *graphAPI) SendGraph(ctx context.Context) error {
	return api.b.SendGraph(ctx)
}
