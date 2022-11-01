package emitter

import (
	"context"
	"math/big"
	"sync"

	"github.com/artnoi43/gsl/concurrent"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/artnoi43/superwatcher/internal/domain/usecase/emitter/reorg"
	"github.com/artnoi43/superwatcher/internal/lib/utils"
	"github.com/artnoi43/superwatcher/pkg/logger"
	"github.com/artnoi43/superwatcher/pkg/superwatcher"
)

// filterLogs filters Ethereum event logs from fromBlock to toBlock,
// and sends *types.Log and *lib.BlockInfo through w.logChan and w.reorgChan respectively.
// If an error is encountered, filterLogs returns with error.
// filterLogs should not be the one sending the error through w.errChan.
func (e *emitter) filterLogs(
	ctx context.Context,
	fromBlock uint64,
	toBlock uint64,
) error {
	var wg sync.WaitGroup
	var mut sync.Mutex
	var eventLogs []types.Log
	var err error

	headersByBlockNumber := make(map[uint64]*types.Header)
	getErrChan := make(chan error)

	// getLogs calls FilterLogs from fromBlock to toBlock
	getLogs := func() {
		eventLogs, err = utils.RetryWithReturn(func() ([]types.Log, error) {
			return e.client.FilterLogs(ctx, ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(fromBlock)),
				ToBlock:   big.NewInt(int64(toBlock)),
				Addresses: e.addresses,
				Topics:    e.topics,
			})
		})

		if err != nil {
			// TODO: what the actual fuck?
			getErrChan <- wrapErrBlockNumber(err, errFetchLogs, (fromBlock+toBlock)/2)
		}
	}

	// getHeader gets block header for a blockNumber
	getHeader := func(blockNumber uint64) {
		header, err := utils.RetryWithReturn(func() (*types.Header, error) {
			return e.client.HeaderByNumber(ctx, big.NewInt(int64(blockNumber)))
		})
		if err != nil {
			getErrChan <- wrapErrBlockNumber(err, errFetchHeader, blockNumber)
		}

		mut.Lock()
		headersByBlockNumber[blockNumber] = header
		mut.Unlock()
	}

	// Get fresh logs, and block headers (fromBlock-toBlock)
	// to compare the headers with that of w.tracker's to detect chain reorg

	wg.Add(1)
	go func() {
		defer wg.Done()
		getLogs()
	}()

	for i := fromBlock; i <= toBlock; i++ {
		wg.Add(1)
		go func(blockNumber uint64) {
			defer wg.Done()
			getHeader(blockNumber)
		}(i)
	}

	// Wait here for logs and headers
	if err := concurrent.WaitAndCollectErrors(&wg, getErrChan); err != nil {
		logger.Error("get fresh data from blockchain failed", zap.Error(err))
		return errors.Wrap(err, "get blockchain data")
	}

	lenLogs := len(eventLogs)
	e.debugMsg("got event logs", zap.Int("number of filtered logs", lenLogs))
	e.debugMsg("got headers and logs", zap.Uint64("fromBlock", fromBlock), zap.Uint64("toBlock", toBlock))

	// Clear all tracker's blocks before fromBlock - lookBackBlocks
	e.debugMsg("clearing tracker", zap.Uint64("clearUntil", fromBlock-e.config.LookBackBlocks))
	e.tracker.ClearUntil(fromBlock - e.config.LookBackBlocks)

	/* Use code from reorg package to manage/handle chain reorg */
	// Use fresh hashes and fresh logs to populate these 3 maps
	freshHashesByBlockNumber, freshLogsByBlockNumber, processLogsByBlockNumber := reorg.PopulateInitialMaps(eventLogs, headersByBlockNumber)
	// wasReorged maps block numbers whose fresh hash and tracker hash differ
	wasReorged := reorg.ProcessReorged(
		e.tracker,
		fromBlock,
		toBlock,
		freshHashesByBlockNumber,
		freshLogsByBlockNumber,
		processLogsByBlockNumber,
	)

	e.debugMsg("wasReorged", zap.Any("wasReorged", wasReorged))

	// If fromBlock was reorged, then return to loopFilterLogs
	if wasReorged[fromBlock] {
		return errors.Wrapf(errFromBlockReorged, "fromBlock %d was removed (chain reorganization)", fromBlock)
	}

	filterResult := new(superwatcher.FilterResult)
	// Publish log(s) and reorged block, and add canon block to tracker
	for blockNumber := fromBlock; blockNumber <= toBlock; blockNumber++ {
		if wasReorged[blockNumber] {
			logger.Info(
				"emitter: chain reorg detected",
				zap.Uint64("blockNumber", blockNumber),
				zap.String("freshHash", freshHashesByBlockNumber[blockNumber].String()),
			)

			reorgedBlock, foundInTracker := e.tracker.GetTrackerBlockInfo(blockNumber)
			if !foundInTracker {
				logger.Panic(
					"blockInfo marked as reorged but was not found in tracker",
					zap.Uint64("blockNumber", blockNumber),
					zap.String("freshHash", reorgedBlock.String()),
				)
			}

			filterResult.ReorgedBlocks = append(filterResult.ReorgedBlocks, reorgedBlock)
			continue
		}

		// Populate blockInfo with fresh info
		// Old, unreorged blocks will not be added to filterResult.GoodBlocks
		b := superwatcher.NewBlockInfo(blockNumber, freshHashesByBlockNumber[blockNumber])
		b.Logs = freshLogsByBlockNumber[blockNumber]

		// Publish block with > 0 block
		if len(b.Logs) > 0 {
			filterResult.GoodBlocks = append(filterResult.GoodBlocks, b)
		}
		// Add ONLY CANONICAL block into tracker
		e.tracker.AddTrackerBlock(b)
	}

	// ToBlock will be the last good block (not reorged)
	filterResult.FromBlock = fromBlock
	if len(wasReorged) == 0 {
		filterResult.LastGoodBlock = toBlock
	} else {
		if l := len(filterResult.GoodBlocks); l == 0 {
			// If there's no block with interesting log in this loop,
			// then filter this whole range again in the next loop.
			filterResult.LastGoodBlock = fromBlock
		} else {
			// Use last good block number as ToBlock
			filterResult.LastGoodBlock = filterResult.GoodBlocks[l-1].Number
		}
	}

	// Publish filterResult via e.filterResultChan
	e.emitFilterResult(filterResult)

	// End loop
	e.debugMsg(
		"number of logs published by filterLogs",
		zap.Int("eventLogs (filtered)", lenLogs),
		zap.Int("processLogs (all logs processed)", len(processLogsByBlockNumber)),
		zap.Int("goodBlocks", len(filterResult.GoodBlocks)),
		zap.Int("reorgedBlocks", len(filterResult.ReorgedBlocks)),
	)

	// Waits until engine syncs
	<-e.syncChan
	e.debugMsg("engine and emitter synced")

	return nil
}