package superwatcher

import (
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
)

// EthClient defines all *ethclient.Client methods used in superwatcher.
// To use a normal *ethclient.Client with superwatcher, wrap it first
// with ethclientwrapper.WrapEthClient.
type EthClient interface {
	BlockNumber(context.Context) (uint64, error)
	FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error)
}
