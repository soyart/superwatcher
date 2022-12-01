package emitter

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/artnoi43/gsl/concurrent"
	"github.com/artnoi43/gsl/gslutils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/pkg/logger"
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
	var eventLogs []types.Log
	var err error

	getErrChan := make(chan error)

	// getLogs calls FilterLogs from fromBlock to toBlock
	getLogs := func() {
		eventLogs, err = gslutils.RetryWithReturn(
			fmt.Sprintf("getLogs from %d to %d", fromBlock, toBlock),

			func() ([]types.Log, error) {
				// No error wrap because in retry mode
				return e.client.FilterLogs(ctx, ethereum.FilterQuery{ //nolint:wrapcheck
					FromBlock: big.NewInt(int64(fromBlock)),
					ToBlock:   big.NewInt(int64(toBlock)),
					Addresses: e.addresses,
					Topics:    e.topics,
				})
			},

			gslutils.Attempts(10),
			gslutils.Delay(4),
			gslutils.LastErrorOnly(true),
		)

		if err != nil {
			// TODO: what the actual fuck?
			getErrChan <- wrapErrBlockNumber(fromBlock, err, errFetchLogs)
		}
	}

	// Get fresh logs, and block headers (fromBlock-toBlock)
	// to compare the headers with that of w.tracker's to detect chain reorg

	wg.Add(1)
	go func() {
		defer wg.Done()
		getLogs()
	}()

	// Wait here for logs and headers
	if err := concurrent.WaitAndCollectErrors(&wg, getErrChan); err != nil {
		e.debugger.Debug(1, "get fresh data from blockchain failed", zap.Error(err))
		return errors.Wrap(err, "get blockchain data")
	}

	lenLogs := len(eventLogs)
	e.debugger.Debug(2, "got headers and logs", zap.Int("logs", lenLogs))

	// Clear all tracker's blocks before fromBlock - filterRange
	until := fromBlock - e.conf.FilterRange
	e.debugger.Debug(2, "clearing tracker", zap.Uint64("untilBlock", until))
	e.tracker.clearUntil(until)

	/* Use code from reorg package to manage/handle chain reorg */
	// Use fresh hashes and fresh logs to populate these 3 maps
	mapFreshHashes, mapFreshLogs, mapProcessLogs := mapFreshLogs(eventLogs)

	// reorgedBlocks maps block numbers whose fresh hash and tracker hash differ, i.e. reorged blocks
	reorgedBlocks, err := processReorg(
		e.tracker,
		fromBlock,
		toBlock,
		mapFreshHashes,
		mapFreshLogs,
		mapProcessLogs,
	)
	if err != nil {
		return errors.Wrap(err, "error detecting chain reorg")
	}

	// Fills |filterResult| and saves current data back to tracker first.
	filterResult := new(superwatcher.FilterResult)
	for blockNumber := fromBlock; blockNumber <= toBlock; blockNumber++ {

		// Reorged blocks (the ones that were removed) will be published with data from tracker
		if reorgedBlocks[blockNumber] {
			reorgedBlock, foundInTracker := e.tracker.getTrackerBlockInfo(blockNumber)
			if !foundInTracker {
				logger.Panic(
					"blockInfo marked as reorged but was not found in tracker",
					zap.Uint64("blockNumber", blockNumber),
					zap.String("freshHash", reorgedBlock.String()),
				)
			}

			logger.Info(
				"chain reorg detected",
				zap.Uint64("blockNumber", blockNumber),
				zap.String("freshHash", mapFreshHashes[blockNumber].String()),
				zap.String("trackerHash", reorgedBlock.String()),
			)

			// ReorgedBlocks field contains the seen, removed blocks from tracker.
			// The new reroged blocks were added to |filterResult| as new GoodBlocks.
			// This means that the engine should process ReorgedBlocks first to revert the TXs,
			// before processing the new, reorged logs.
			filterResult.ReorgedBlocks = append(filterResult.ReorgedBlocks, reorgedBlock)
		}

		// New blocks will use fresh information. This includes new block after a reorg.
		logs, ok := mapFreshLogs[blockNumber]
		if !ok {
			continue
		}
		hash, ok := mapFreshHashes[blockNumber]
		if !ok {
			return errors.Wrapf(errNoHash, "blockNumber %d", blockNumber)
		}

		// Only add fresh, canonical blockInfo with interesting logs
		b := &superwatcher.BlockInfo{
			Number: blockNumber,
			Hash:   hash,
			Logs:   logs,
		}
		b.Logs = mapFreshLogs[blockNumber]
		e.tracker.addTrackerBlockInfo(b)

		// Publish only block with logs
		if b.Logs != nil {
			filterResult.GoodBlocks = append(filterResult.GoodBlocks, b)
		}
	}

	// If fromBlock was reorged, then return to loopFilterLogs.
	// Results will not be published, so the engine will never know that fromBlock is reorging.
	// **The reorged blocks different hashes have also been saved to tracker,
	// so if they come back in the next loop, with the same hash here, they'll be marked as non-reorg blocks**.
	if reorgedBlocks[fromBlock] {
		return errors.Wrapf(errFromBlockReorged, "fromBlock %d was removed (chain reorganization)", fromBlock)
	}

	filterResult.FromBlock = fromBlock
	filterResult.ToBlock = toBlock

	// TODO: Test this
	// Decide result.LastGoodBlock
	if len(reorgedBlocks) == 0 {
		// If no reorg, just use toBlock
		filterResult.LastGoodBlock = toBlock

		// If reorg (there should be goodBlocks too)
	} else {
		// If there's also goodBlocks during reorg
		if l := len(filterResult.GoodBlocks); l != 0 {
			// Use last good block's number as LastGoodBlock
			lastGood := filterResult.GoodBlocks[l-1].Number
			firstReorg := filterResult.ReorgedBlocks[0].Number

			if lastGood > firstReorg {
				lastGood = firstReorg - 1
			}

			filterResult.LastGoodBlock = lastGood

		} else {
			filterResult.LastGoodBlock = fromBlock
		}
	}

	// Publish filterResult via e.filterResultChan
	e.emitFilterResult(filterResult)

	// End loop
	e.debugger.Debug(
		1, "number of logs published by filterLogs",
		zap.Int("eventLogs (filtered)", lenLogs),
		zap.Int("processLogs (all logs processed)", len(mapProcessLogs)),
		zap.Int("goodBlocks", len(filterResult.GoodBlocks)),
		zap.Int("reorgedBlocks", len(filterResult.ReorgedBlocks)),
		zap.Uint64("lastGoodBlock", filterResult.LastGoodBlock),
	)

	// Waits until engine syncs
	e.SyncsWithEngine()

	return nil
}
