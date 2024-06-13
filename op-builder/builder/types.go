package builder

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	INIT = iota
	EXEC = iota
)

type Message struct {
	ChainId *big.Int       `json:"chainId"`
	Data    hexutil.Bytes  `json:"data"`
	Target  common.Address `json:"target"`
	Value   *big.Int       `json:"value"`
}

type CrossBundle struct {
	Messages []Message `json:"messsages"`
}
