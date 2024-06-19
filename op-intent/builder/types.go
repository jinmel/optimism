package builder

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Message struct {
	ChainId *big.Int       `json:"chainId"`
	Data    hexutil.Bytes  `json:"data"`
	Target  common.Address `json:"target"`
	Value   *big.Int       `json:"value"`
}

type GraphBundle struct {
	Root  Message       `json:"root"`
	Nodes []GraphBundle `json:"nodes"`
}
