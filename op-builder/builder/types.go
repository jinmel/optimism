package builder

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	INIT = iota
	EXEC = iota
)

type Message struct {
	ChainId *big.Int       `json:"chainId"`
	Data    []byte         `json:"data"`
	Target  common.Address `json:"target"`
	Value   *big.Int       `json:"value"`
	Payload []byte         `json:"payload"`
}

type CrossBundle struct {
	Messages []*Message `json:"messsage"`
}
