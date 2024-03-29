package poller

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/soyart/gsl"
	"go.uber.org/zap"

	"github.com/soyart/superwatcher"
	"github.com/soyart/superwatcher/pkg/logger/debugger"
)

// mapLogsResult represents information of fresh blocks mapped by mapLogs.
// It contains fresh data, i.e. not from tracker.
type mapLogsResult struct {
	forked bool // true if the tracker block hash differs from fresh block hash
	superwatcher.Block
}

type param struct {
	fromBlock uint64
	toBlock   uint64
	policy    superwatcher.Policy
}

func (p *poller) Poll(
	ctx context.Context,
	fromBlock uint64,
	toBlock uint64,
) (
	*superwatcher.PollerResult,
	error,
) {
	p.Lock()
	defer p.Unlock()

	if p.tracker != nil || p.lastRecordedBlock != 0 {
		// Clear all tracker's blocks before fromBlock - filterRange
		until := p.lastRecordedBlock - p.filterRange
		p.debugger.Debug(2, "clearing tracker", zap.Uint64("untilBlock", until))
		p.tracker.clearUntil(until)
	}

	param := &param{
		fromBlock: fromBlock,
		toBlock:   toBlock,
		policy:    p.policy,
	}

	pollResults, err := poll(ctx, param, p.addresses, p.topics, p.client, p.debugger)
	if err != nil {
		return nil, err
	}

	var blocksMissing []uint64
	pollResults, blocksMissing, err = pollMissing(ctx, param, p.client, p.tracker, pollResults, p.debugger)
	if err != nil {
		return nil, err
	}

	pollResults, err = findReorg(param, blocksMissing, p.tracker, pollResults, p.debugger)
	if err != nil {
		return nil, err
	}

	result, err := processResult(param, p.tracker, pollResults, p.debugger)
	if err != nil {
		return result, err
	}

	result.FromBlock, result.ToBlock = fromBlock, toBlock
	result.LastGoodBlock = superwatcher.LastGoodBlock(result)
	p.lastRecordedBlock = result.LastGoodBlock

	fromBlockResult, ok := pollResults[fromBlock]
	if ok {
		if fromBlockResult.forked && p.doReorg {
			return result, errors.Wrapf(
				superwatcher.ErrFromBlockReorged, "fromBlock %d was removed/reorged", fromBlock,
			)
		}
	}

	return result, nil
}

func poll(
	ctx context.Context,
	param *param,
	addresses []common.Address,
	topics [][]common.Hash,
	client superwatcher.EthClient,
	debugger *debugger.Debugger,
) (
	map[uint64]*mapLogsResult,
	error,
) {
	pollResults := make(map[uint64]*mapLogsResult)

	switch {
	case param.policy >= superwatcher.PolicyExpensiveBlock: // Get blocks and event logs concurrently
		return nil, errors.New("PolicyExpensiveBlock not implemented")

	case param.policy == superwatcher.PolicyExpensive:
		return pollExpensive(ctx, param.fromBlock, param.toBlock, addresses, topics, client, pollResults, debugger)

	case param.policy <= superwatcher.PolicyNormal:
		return pollCheap(ctx, param.fromBlock, param.toBlock, addresses, topics, client, pollResults, debugger)
	}

	panic(superwatcher.ErrBadPolicy.Error() + " " + param.policy.String())
}

// pollMissing polls tracker blocks that were removed/reorged and thus currently missing from the chain.
func pollMissing(
	ctx context.Context,
	param *param,
	client superwatcher.EthClient,
	tracker *blockTracker, // tracker is used as read-only in here. Don't write.
	pollResults map[uint64]*mapLogsResult,
	debugger *debugger.Debugger,
) (
	map[uint64]*mapLogsResult,
	[]uint64, // tracker's blocks that went missing (no logs)
	error,
) {
	if tracker == nil {
		return pollResults, nil, nil
	}

	// Find missing blocks (blocks in tracker that are not in pollResults)
	var blocksMissing []uint64
	for n := param.toBlock; n >= param.fromBlock; n-- {
		// Not in tracker => not missing
		if _, ok := tracker.getTrackerBlock(n); !ok {
			continue
		}

		// In tracker and in pollResults => not missing
		if _, ok := pollResults[n]; ok {
			continue
		}

		blocksMissing = append(blocksMissing, n)
	}

	lenBlocks := len(blocksMissing)

	if lenBlocks == 0 {
		debugger.Debug(3, "found no missing blocks")
		return pollResults, blocksMissing, nil
	}

	debugger.Debug(
		1, fmt.Sprintf("found %d blocks missing, getting their headers", lenBlocks),
		zap.Uint64s("blocksMissing", blocksMissing),
	)

	headers, err := getHeadersByNumbers(ctx, client, blocksMissing)
	if err != nil {
		return nil, blocksMissing, errors.Wrap(superwatcher.ErrFetchError, "failed to get block headers in mapLogsNg")
	}

	lenHeads := len(headers)
	if lenHeads != lenBlocks {
		return nil, blocksMissing, errors.Wrapf(
			superwatcher.ErrFetchError, "expecting %d headers, got %d", lenBlocks, lenHeads,
		)
	}

	// Collect headers for blocksMissing
	_, err = collectHeaders(pollResults, param.fromBlock, param.toBlock, headers)
	if err != nil {
		if errors.Is(err, errHashesDiffer) {
			// deleteMapResults(pollResults, lastGood)
			return pollResults, blocksMissing, err
		}

		return nil, nil, errors.Wrap(err, "collectHeaders error")
	}

	return pollResults, blocksMissing, nil
}

// findReorg compares fresh block hashes with known hashes in tracker.
// If block hashes and logs length do not match, findReorg marks the block as reorged.
func findReorg(
	param *param,
	blocksMissing []uint64,
	tracker *blockTracker,
	pollResults map[uint64]*mapLogsResult,
	debugger *debugger.Debugger,
) (
	map[uint64]*mapLogsResult,
	error,
) {
	if tracker == nil {
		return pollResults, nil
	}

	// Detect chain reorg using tracker
	for n := param.fromBlock; n <= param.toBlock; n++ {
		trackerBlock, ok := tracker.getTrackerBlock(n)
		if !ok {
			continue
		}

		pollResult, ok := pollResults[n]
		if !ok {
			// Not in result, but in range + in tracker => kinda sus
			return nil, errors.Wrapf(superwatcher.ErrProcessReorg, "pollResult missing for trackerBlock %d", n)
		}

		if trackerBlock.Hash == pollResult.Hash && len(trackerBlock.Logs) == len(pollResult.Logs) {
			continue
		}

		if gsl.Contains(blocksMissing, n) {
			pollResult.LogsMigrated = true
		}

		debugger.Debug(
			1, "chain reorg detected",
			zap.Uint64("blockNumber", n),
			zap.String("freshHash", pollResult.String()),
			zap.String("trackerHash", trackerBlock.String()),
			zap.Int("freshLogs", len(pollResult.Logs)),
			zap.Int("trackerLogs", len(trackerBlock.Logs)),
		)

		pollResult.forked = true
	}

	return pollResults, nil
}

// processResult collects poll results from |tracker| and |pollResults| into superwatcher.PollerResult.
// It collects the result while also removing/adding fresh blocks to tracker, as per param.Policy.
// When collecting, it copies superwatcher.Block values into PollerResult to avoid mutating tracker values.
func processResult(
	param *param,
	tracker *blockTracker,
	pollResults map[uint64]*mapLogsResult,
	debugger *debugger.Debugger,
) (
	*superwatcher.PollerResult,
	error,
) {
	// Fills |result| and saves current data back to tracker first.
	result := new(superwatcher.PollerResult)
	for number := param.fromBlock; number <= param.toBlock; number++ {
		// Only blocks in pollResults are worth processing.
		// There are 3 reasons a block is in pollResults:
		// (1) block has >=1 interesting log
		// (2) block _did_ have >= logs from the last call, but was reorged and no longer has any interesting logs
		// If (2), then it will removed from tracker, and will no longer appear in pollResults after this call.

		pollResult, ok := pollResults[number]
		if !ok {
			continue
		}

		// Reorged blocks (the ones that were removed) will be published with data from tracker
		if pollResult.forked && tracker != nil {
			trackerBlock, ok := tracker.getTrackerBlock(number)
			if !ok || trackerBlock == nil {
				debugger.Debug(
					1, "block marked as reorged but was not found (or nil) in tracker",
					zap.Uint64("blockNumber", number),
					zap.String("freshHash", trackerBlock.String()),
				)

				return nil, errors.Wrapf(
					superwatcher.ErrProcessReorg, "reorgedBlock %d not found in trackfromBlocker", number,
				)
			}

			// Logs may be moved from blockNumber, hence there's no value in map
			freshHash := pollResult.Hash

			// Copy to avoid mutated trackerBlock which might break poller logic.
			// After the copy, result.ReorgedBlocks consumer may freely mutate their *Block.
			copiedFromTracker := *trackerBlock
			result.ReorgedBlocks = append(result.ReorgedBlocks, &copiedFromTracker)

			// Block used to have interesting logs, but chain reorg occurred
			// and its logs were moved to somewhere else, or just removed altogether.
			if pollResult.LogsMigrated {
				debugger.Debug(
					1, "logs missing from block",
					zap.Uint64("blockNumber", number),
					zap.String("freshHash", freshHash.String()),
					zap.String("trackerHash", trackerBlock.String()),
					zap.Int("old logs", len(trackerBlock.Logs)),
				)

				err := handleBlocksMissingPolicy(number, tracker, trackerBlock, freshHash, param.policy)
				if err != nil {
					return nil, errors.Wrap(superwatcher.ErrProcessReorg, err.Error())
				}
			}
		}

		freshBlock := pollResult.Block
		addTrackerBlockPolicy(tracker, &freshBlock, param.policy)

		// Copy goodBlock to avoid poller users mutating goodBlock values inside of tracker.
		goodBlock := freshBlock
		result.GoodBlocks = append(result.GoodBlocks, &goodBlock)
	}

	return result, nil
}

// handleBlocksMissingPolicy handles blocks that is marked with LogsMigrated (0 logs)
func handleBlocksMissingPolicy(
	number uint64,
	tracker *blockTracker,
	trackerBlock *superwatcher.Block,
	freshHash common.Hash,
	policy superwatcher.Policy,
) error {
	switch {
	case policy == superwatcher.PolicyFast:
		// Remove from tracker if block has 0 logs, and poller will cease to
		// get block header for this empty block after this call.
		if err := tracker.removeBlock(number); err != nil {
			return errors.Wrap(superwatcher.ErrProcessReorg, err.Error())
		}

	default:
		// Save new empty block information back to tracker. This will make poller
		// continues to get header for this block until it goes out of filter (poll) scope.
		trackerBlock.Hash = freshHash
		trackerBlock.Logs = nil
	}

	return nil
}

// addTrackerBlockPolicy adds blocks to tracker based on PollPolicy.
// PolicyCheap will not save blocks with 0 logs to tracker, so as to avoid expensive header fetching.
func addTrackerBlockPolicy(tracker *blockTracker, block *superwatcher.Block, policy superwatcher.Policy) {
	if tracker == nil {
		return
	}
	if policy == superwatcher.PolicyFast {
		if len(block.Logs) == 0 {
			return
		}
	}

	tracker.addTrackerBlock(block)
}
