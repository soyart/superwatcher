package poller

import (
	"context"
	"math/big"
	"sync"

	"github.com/artnoi43/gsl/concurrent"
	"github.com/artnoi43/superwatcher/pkg/logger/debugger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/artnoi43/superwatcher"
)

// pollExpensive concurrently fetches event logs and block headers
// for all blocks within range [fromBlock, toBlock] and save them in pollResults.
func pollExpensive(
	ctx context.Context,
	fromBlock uint64,
	toBlock uint64,
	addresses []common.Address,
	topics [][]common.Hash,
	client superwatcher.EthClient,
	pollResults map[uint64]*mapLogsResult,
	debugger *debugger.Debugger,
) (
	map[uint64]*mapLogsResult,
	error,
) {
	var headers map[uint64]superwatcher.BlockHeader
	var logs []types.Log

	var blockNumbers []uint64
	for n := fromBlock; n <= toBlock; n++ {
		blockNumbers = append(blockNumbers, n)
	}

	q := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: addresses,
		Topics:    topics,
	}
	// Get block headers and event logs concurrently
	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(2) // Get headers and logs concurrently in 2 Goroutines

	go func() {
		defer wg.Done()
		var err error
		logs, err = client.FilterLogs(ctx, q)
		if err != nil {
			errChan <- errors.Wrapf(superwatcher.ErrFetchError, "filterLogs returned error")
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		headers, err = getHeadersByNumbers(ctx, client, blockNumbers)
		if err != nil {
			errChan <- errors.Wrapf(err, "getHeadersByNumbers returned error")
		}
	}()

	if err := concurrent.WaitAndCollectErrors(&wg, errChan); err != nil {
		return nil, errors.Wrap(err, "concurrent fetch error")
	}

	debugger.Debug(
		2, "polled event logs and headers",
		zap.Int("logs", len(logs)),
		zap.Int("headers", len(headers)),
	)

	if len(blockNumbers) != len(headers) {
		return nil, errors.Wrap(superwatcher.ErrFetchError, "headers and blockNumbers length not matched")
	}

	// Collect logs into map
	_, err := collectLogs(pollResults, logs)
	if err != nil {
		if errors.Is(err, errHashesDiffer) {
			// deleteMapResults(pollResults, lastGood)
			return pollResults, err
		}

		return nil, errors.Wrap(err, "collectLogs found error")
	}

	_, err = collectHeaders(pollResults, fromBlock, toBlock, headers)
	if err != nil {
		if errors.Is(err, errHashesDiffer) {
			// deleteMapResults(pollResults, lastGood)
			return pollResults, err
		}

		return nil, errors.Wrap(err, "collectHeaders found error")
	}

	debugger.Debug(3, "pollExpensive successful")
	return pollResults, nil
}

// pollCheap polls event logs first, and then block headers for blocks with logs.
// The results will be written to pollResults.
func pollCheap(
	ctx context.Context,
	fromBlock uint64,
	toBlock uint64,
	addresses []common.Address,
	topics [][]common.Hash,
	client superwatcher.EthClient,
	pollResults map[uint64]*mapLogsResult,
	debugger *debugger.Debugger,
) (
	map[uint64]*mapLogsResult,
	error,
) {
	q := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: addresses,
		Topics:    topics,
	}

	// Just get event logs for now
	logs, err := client.FilterLogs(ctx, q)
	if err != nil {
		return nil, errors.Wrap(superwatcher.ErrFetchError, err.Error())
	}

	debugger.Debug(2, "polled event logs", zap.Int("len", len(logs)))

	_, err = collectLogs(pollResults, logs)
	if err != nil {
		if errors.Is(err, errHashesDiffer) {
			return pollResults, err
		}

		return nil, errors.Wrap(err, "collectLogs error")
	}

	var targetBlocks []uint64
	for n := fromBlock; n <= toBlock; n++ {
		// PolicyFast only fetch headers for blocks in pollResults
		if _, ok := pollResults[n]; !ok {
			continue
		}

		targetBlocks = append(targetBlocks, n)
	}

	debugger.Debug(
		3, "polling headers for targetBlocks",
		zap.Uint64s("targetBlocks", targetBlocks),
	)

	headers, err := getHeadersByNumbers(ctx, client, targetBlocks)
	if err != nil {
		return nil, errors.Wrap(superwatcher.ErrFetchError, "failed to get headers for resultBlocks")
	}

	_, err = collectHeaders(pollResults, fromBlock, toBlock, headers)
	if err != nil {
		if errors.Is(err, errHashesDiffer) {
			return pollResults, err
		}

		return nil, errors.Wrap(err, "collectHeaders error")
	}

	debugger.Debug(3, "pollCheap successful")
	return pollResults, nil
}
