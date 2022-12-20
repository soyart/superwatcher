package reorgsim

import (
	"fmt"

	"github.com/artnoi43/gsl/gslutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Use this as ReorgEvent.ReorgBlock to disable chain reorg.
const NoReorg uint64 = 0

type blockChain map[uint64]*block

// MoveLogs represent a move of logs to a new blockNumber
type MoveLogs struct {
	NewBlock uint64
	TxHashes []common.Hash // txHashes of logs to be moved to newBlock
}

// reorg calls `*block.reorg` on every block whose blockNumber is greater than |reorgedAt|.
// Unlike `*block.reorg`, which returns a `*block`, c.reorg(reorgedAt) modifies c in-place.
func (c blockChain) reorg(reorgedBlock uint64) {
	for number, block := range c {
		if number >= reorgedBlock {
			c[number] = block.reorg()
		}
	}
}

// moveLogs moved will have their blockHash and blockNumber changed to destination blocks.
// If you are manually reorging and moving logs, call blockChain.reorg before blockChain.moveLogs.
// If you are creating new reorged chain with moved logs, use NewBlockChainReorgMoveLogs instead,
// as blockChain.moveLogs only moves the logs and changing log.BlockHash and log.BlockNumber.
// It also returns 2 slices of block numbers, 1st of which is a slice of blocks from which logs are moved,
// the 2nd of which is a slice of blocks to which logs are moved.
// NOTE: Do not use this function directly, since it only moves logs to new blocks and does not reorg blocks.
// It is meant to be used inside NewBlockChainReorgMoveLogs, and NewBlockChain
func (c blockChain) moveLogs(
	movedLogs map[uint64][]MoveLogs,
) (
	[]uint64, // Blocks from which logs are moved from
	[]uint64, // Blocks to which logs art moved to
) {
	// A slice of unique blockNumbers that logs will be moved to.
	// Might be useful to caller, maybe to create empty blocks (no logs) for the old chain.
	var moveFromBlocks []uint64
	var moveToBlocks []uint64

	for moveFromBlock, moves := range movedLogs {
		b, ok := c[moveFromBlock]
		if !ok {
			panic(fmt.Sprintf("logs moved from non-existent block %d", moveFromBlock))
		}

		moveFromBlocks = append(moveFromBlocks, moveFromBlock)

		for _, move := range moves {
			targetBlock, ok := c[move.NewBlock]
			if !ok {
				panic(fmt.Sprintf("logs moved to non-existent block %d", move.NewBlock))
			}

			// Add unique moveToBlocks
			if !gslutils.Contains(moveToBlocks, move.NewBlock) {
				moveToBlocks = append(moveToBlocks, move.NewBlock)
			}

			// Save logsToMove before removing it from b
			var logsToMove []types.Log
			for _, log := range b.logs {
				if gslutils.Contains(move.TxHashes, log.TxHash) {
					logsToMove = append(logsToMove, log)
				}
			}

			// Remove logs from b
			b.removeLogs(move.TxHashes)

			// Change log.BlockHash to new BlockHash
			for i := range logsToMove {
				logsToMove[i].BlockNumber = targetBlock.blockNumber
				logsToMove[i].BlockHash = targetBlock.hash
			}

			targetBlock.logs = append(targetBlock.logs, logsToMove...)
		}
	}

	return moveFromBlocks, moveToBlocks
}

// newBlockChain returns a new blockChain from |mappedLogs|. The parameter |reorgedAt|
// is used to determine block.reorgedHere and block.toBeForked
func newBlockChain(
	mappedLogs map[uint64][]types.Log,
	reorgedBlock uint64,
) blockChain {
	chain := make(blockChain)

	var noReorg bool
	if reorgedBlock == NoReorg {
		noReorg = true
	}

	for blockNumber, logs := range mappedLogs {
		var toBeForked bool
		if noReorg {
			toBeForked = false
		} else {
			toBeForked = blockNumber >= reorgedBlock
		}

		chain[blockNumber] = &block{
			blockNumber: blockNumber,
			hash:        logs[0].BlockHash,
			logs:        logs,
			reorgedHere: blockNumber == reorgedBlock,
			toBeForked:  toBeForked,
		}
	}

	return chain
}

// NewBlockChain is the preferred way to init reorgsim `blockChain`s. It accept a slice of `ReorgEvent` and
// uses each event to construct a reorged chain, which will be appended to the second return variable.
// Each ReorgEvent will result in its own blockChain, with the identical index.
func NewBlockChain(
	logs map[uint64][]types.Log,
	events []ReorgEvent,
) (
	blockChain, // Original chain
	[]blockChain, // Reorged chains
) {
	if len(events) == 0 {
		return newBlockChain(logs, NoReorg), nil
	}

	chain := newBlockChain(logs, events[0].ReorgBlock)
	var reorgedChains = make([]blockChain, len(events))

	for i, event := range events {
		var prevChain blockChain
		if i == 0 {
			prevChain = chain
		} else {
			prevChain = reorgedChains[i-1]
		}

		// Reorg and move logs
		forkedChain := copyBlockChain(prevChain)
		forkedChain.reorg(event.ReorgBlock)
		moveFromBlocks, moveToBlocks := forkedChain.moveLogs(event.MovedLogs)

		// Ensure that all moveFromBlock exist in forkedChain.
		// If the forkedChain did not have moveToBlock, create one.
		// This created block will need to have non-deterministic blockHash via RandomHash()
		// because the block needs to have different blockHash vs the reorgedBlock's hash (PRandomHash()).
		for _, prevFrom := range moveFromBlocks {
			if _, ok := prevChain[prevFrom]; !ok {
				panic(fmt.Sprintf("moved from non-existent block %d in the old chain", prevFrom))
			}

			if b, ok := forkedChain[prevFrom]; !ok || b == nil {
				forkedChain[prevFrom] = &block{
					blockNumber: prevFrom,
					hash:        RandomHash(prevFrom),
					reorgedHere: prevFrom == event.ReorgBlock,
					toBeForked:  true,
				}
			}
		}

		// Ensure that all moveToBlocks exist in prevChain.
		// If the old chain did not have moveToBlock, create one.
		// This created block will need to have non-deterministic blockHash via RandomHash()
		// because the block needs to have different blockHash vs the reorgedBlock's hash (PRandomHash()).
		for _, forkedTo := range moveToBlocks {
			if _, ok := forkedChain[forkedTo]; !ok {
				panic(fmt.Sprintf("moved to non-existent block %d in the new chain", forkedTo))
			}

			if _, ok := prevChain[forkedTo]; !ok {
				prevChain[forkedTo] = &block{
					blockNumber: forkedTo,
					hash:        RandomHash(forkedTo),
					reorgedHere: forkedTo == event.ReorgBlock,
					toBeForked:  true,
				}
			}
		}

		// Make sure the block from which the logs moved
		reorgedChains[i] = forkedChain
	}

	return chain, reorgedChains
}

// newBlockChainReorgSimple returns a tuple of 2 blockChain, the first being the original chain,
// and the second being the reorged chain based solely on 1 |reorgedBlock| (no logs will be moved to new blocks).
// It is unexported and is only here as a convenient function wrapped by others, and for reorg logic tests.
// NOTE: maybe deprecated in favor of NewBlockChain
func newBlockChainReorgSimple(
	mappedLogs map[uint64][]types.Log,
	reorgedBlock uint64,
) (
	blockChain,
	blockChain,
) {
	// The "good old chain"
	chain := newBlockChain(mappedLogs, reorgedBlock)

	// No reorg - use the same chain
	if reorgedBlock == NoReorg {
		return chain, chain
	}

	// |reorgedChain| will differ from |oldChain| after |reorgedAt|
	reorgedChain := copyBlockChain(chain)
	reorgedChain.reorg(reorgedBlock)

	return chain, reorgedChain
}

// NewBlockChainReorgMoveLogs calls newBlockChainReorgSimple, and uses |event.MovedLogs| to move logs.
// If |event.MovedLogs| is nil, the result chains are identical to those of newBlockChainReorgSimple.
// NOTE: maybe deprecated in favor of NewBlockChain
func NewBlockChainReorgMoveLogs(
	mappedLogs map[uint64][]types.Log,
	event ReorgEvent,
) (
	blockChain,
	blockChain,
) {
	for blockNumber := range event.MovedLogs {
		if blockNumber < event.ReorgBlock {
			panic(fmt.Sprintf("blockNumber %d < reorgedAt %d", blockNumber, event.ReorgBlock))
		}
	}

	chain, reorgedChain := newBlockChainReorgSimple(mappedLogs, event.ReorgBlock)

	if len(event.MovedLogs) != 0 {
		_, moveToBlocks := reorgedChain.moveLogs(event.MovedLogs)

		// Ensure that all moveToBlocks exist in original chain
		for _, moveToBlock := range moveToBlocks {
			// If the old chain did not have moveToBlock, create one.
			// This created block will need to have non-deterministic blockHash via RandomHash()
			// because the block needs to have different blockHash vs the reorgedBlock's hash (PRandomHash()).
			if _, ok := chain[moveToBlock]; !ok {
				chain[moveToBlock] = &block{
					blockNumber: moveToBlock,
					hash:        RandomHash(moveToBlock),
					reorgedHere: moveToBlock == event.ReorgBlock,
					toBeForked:  true,
				}
			}
		}
	}

	return chain, reorgedChain
}
