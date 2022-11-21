package emitter

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"testing"

	"github.com/artnoi43/gsl/gslutils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"

	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/config"
	"github.com/artnoi43/superwatcher/pkg/datagateway/watcherstate/mockwatcherstate"
	"github.com/artnoi43/superwatcher/pkg/reorgsim"
)

// TODO: verbose does not work
func getFlagValues() (caseNumber int, verbose bool) {
	caseNumber = 1 // default case 1
	if flagCase != nil {
		caseNumber = *flagCase
	}
	if verboseFlag != nil {
		verbose = *verboseFlag
	}

	return caseNumber, verbose
}

// allCasesAlreadyRun is used to skip TestEmitterByCase if TestEmitterAllCases were run.
var allCasesAlreadyRun bool

func TestEmitterAllCases(t *testing.T) {
	allCasesAlreadyRun = true

	_, verbose := getFlagValues()
	for i := range testCases {
		testName := fmt.Sprintf("Case:%d", i+1)
		t.Run(testName, func(t *testing.T) {
			emitterTestTemplate(t, i+1, verbose)
		})
	}
}

// Run this from the root of the repo with:
// go test -v ./internal/emitter -run TestEmitterByCase -case 69
// Go test binary already called `flag.Parse`, so we just simply
// need to name our flag so that the flag package knows to parse it too.
var flagCase = flag.Int("case", -1, "Emitter test case")
var verboseFlag = flag.Bool("v", false, "Verbose emitter output")

func TestEmitterByCase(t *testing.T) {
	if allCasesAlreadyRun {
		t.Skip("all cases were tested before -- skipping")
	}

	caseNumber, verbose := getFlagValues()
	if caseNumber < 0 {
		TestEmitterAllCases(t)
		return
	}

	if len(testCases) > caseNumber {
		testName := fmt.Sprintf("Case:%d", caseNumber)
		t.Run(testName, func(t *testing.T) {
			emitterTestTemplate(t, caseNumber, verbose)
		})

		return
	}

	t.Skipf("no such test case: %d", caseNumber)
}

func emitterTestTemplate(t *testing.T, caseNumber int, verbose bool) {
	tc := testCases[caseNumber-1]
	b, _ := json.Marshal(tc)
	t.Logf("testConfig for case %d: %s", caseNumber, b)

	conf, _ := config.ConfigYAML("../../config/config.yaml")
	// Override LoopInterval
	conf.LoopInterval = 0

	fakeRedis := mockwatcherstate.New(tc.FromBlock - 1)

	param := reorgsim.ReorgParam{
		StartBlock:    tc.FromBlock,
		BlockProgress: 20,
		ReorgedAt:     tc.ReorgedAt,
		ExitBlock:     tc.ToBlock + 200,
	}

	sim := reorgsim.NewReorgSim(param, tc.LogsFiles)

	// Buffered error channels, because if sim will die on ExitBlock, then it will die multiple times
	errChan := make(chan error, 5)
	syncChan := make(chan struct{})
	filterResultChan := make(chan *superwatcher.FilterResult)

	testEmitter := New(conf, sim, fakeRedis, nil, nil, syncChan, filterResultChan, errChan, verbose)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		testEmitter.(*emitter).Loop(ctx)
	}()

	tracker := newTracker()
	var seenLogs []*types.Log

	go func() {
		if err := <-errChan; err != nil {
			if errors.Is(err, reorgsim.ErrExitBlockReached) {
				// This triggers shutdown on testEmitter, causing result from channels to be nil
				cancel()
			}
		}
	}()

	for {
		result := <-filterResultChan

		if result == nil {
			break
		}
		if result.LastGoodBlock > tc.ToBlock {
			t.Logf("finished case %d", caseNumber)
			cancel()
			break
		}

		lastGoodBlock := result.LastGoodBlock

		for i, block := range result.ReorgedBlocks {
			blockNumber := block.Number
			hash := block.Hash

			if b, exists := tracker.getTrackerBlockInfo(blockNumber); exists {
				if b.Hash == hash {
					t.Fatalf("ReorgedBlocks[%d] is not reorg: hash=%v", i, hash)
				}
			} else {
				t.Fatalf("ReorgedBlocks[%d] didn't check before", i)
			}

			// Check LastGoodBlock
			if lastGoodBlock > blockNumber {
				t.Fatalf(
					"invalid LastGoodBlock: ReorgedBlocks[%d] blockNumber=%v, LastGoodBlock=%v",
					i, blockNumber, lastGoodBlock,
				)
			}

			// Check that all the reorged logs were seen before in |seenLogs|
			for _, log := range block.Logs {
				if !gslutils.Contains(seenLogs, log) {
					fatalBadLog(t, "reorgedLog not seen before", log)
				}
			}

			tracker.addTrackerBlockInfo(block)
		}

		for i, block := range result.GoodBlocks {
			blockNumber := block.Number
			hash := block.Hash

			if b, exists := tracker.getTrackerBlockInfo(blockNumber); exists {
				if b.Hash != hash {
					t.Fatalf(
						"GoodBlocks[%d] is reorged: hash(before)=%v hash(after)=%v",
						i, b.Hash.String(), hash.String(),
					)
				}
			}
			for _, log := range block.Logs {
				var ok bool
				seenLogs, ok = appendUnique(seenLogs, log)
				if !ok {
					fatalBadLog(t, "duplicate good log in seenLogs", log)
				}
			}

			tracker.addTrackerBlockInfo(block)
		}

		// Sets before syncs
		testEmitter.(*emitter).stateDataGateway.SetLastRecordedBlock(ctx, result.LastGoodBlock)
		syncChan <- struct{}{}
	}
}

func fatalBadLog(t *testing.T, msg string, log *types.Log) {
	t.Fatalf(
		"%s: blockHash %s, txHash %s, addr %s, topics[0]: %s",
		msg, log.BlockHash.String(), log.TxHash.String(), log.Address.String(), log.Topics[0].String(),
	)
}

// appendUnique appends item to arr if arr does not contain item.
func appendUnique[T comparable](arr []T, item T) ([]T, bool) {
	if !gslutils.Contains(arr, item) {
		return append(arr, item), true
	}

	return arr, false
}
