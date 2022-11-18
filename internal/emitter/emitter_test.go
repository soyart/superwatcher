package emitter

import (
	"context"
	"testing"

	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/config"
	"github.com/artnoi43/superwatcher/pkg/reorgsim"
)

func TestEmitter(t *testing.T) {
	t.Run("case testEmitterCase1", testEmitterCase1)
	t.Run("case testEmitterCase2", testEmitterCase2)
	t.Run("case testEmitterCase3", testEmitterCase3)
	t.Run("case testEmitterCase4", testEmitterCase4)
	t.Run("case testEmitterCase5", testEmitterCase5)
}

func testEmitterCase1(t *testing.T) {
	tc := testCases[0]
	
	syncChan := make(chan struct{})
	filterResultChan := make(chan *superwatcher.FilterResult)
	errChan := make(chan error)

	conf, _ := config.ConfigYAML("../../config/config.yaml")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fakeDataGateway := &mockStateDataGateway{value: tc.fromBlock}
	sim := reorgsim.NewReorgSim(5, tc.fromBlock-1, tc.reorgedAt, tc.logs)
	e := New(conf, sim, fakeDataGateway, nil, nil, syncChan, filterResultChan, errChan, true)

	go func() {
		e.(*emitter).loopFilterLogs(ctx)
	}()

	tracker := newTracker()

	for {
		result := <-filterResultChan
		lastGoodBlock := result.LastGoodBlock

		// check ReorgedBlocks
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
			tracker.addTrackerBlockInfo(block)

			// check LastGoodBlock
			if blockNumber < lastGoodBlock {
				t.Fatalf("LastGoodBlock is wrong: ReorgedBlocks[%d] blockNumber=%v, LastGoodBlock=%v", i, blockNumber, lastGoodBlock)
			}
		}

		// check GoodBlocks
		for i, block := range result.GoodBlocks {
			blockNumber := block.Number
			hash := block.Hash

			if b, exists := tracker.getTrackerBlockInfo(blockNumber); exists {
				if b.Hash != hash {
					t.Fatalf("GoodBlocks[%d] is reorg: hash(before)=%v hash(after)=%v", i, b.Hash, hash)
				}
			}
			tracker.addTrackerBlockInfo(block)
		}

		syncChan <- struct{}{}
	}
}

func testEmitterCase2(t *testing.T) {
	tc := testCases[1]

	syncChan := make(chan struct{})
	filterResultChan := make(chan *superwatcher.FilterResult)
	errChan := make(chan error)

	conf, _ := config.ConfigYAML("../../config/config.yaml")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fakeDataGateway := &mockStateDataGateway{value: tc.fromBlock}
	sim := reorgsim.NewReorgSim(5, tc.fromBlock-1, tc.reorgedAt, tc.logs)
	e := New(conf, sim, fakeDataGateway, nil, nil, syncChan, filterResultChan, errChan, true)

	go func() {
		e.(*emitter).loopFilterLogs(ctx)
	}()

	tracker := newTracker()

	for {
		result := <-filterResultChan
		lastGoodBlock := result.LastGoodBlock

		// check ReorgedBlocks
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
			tracker.addTrackerBlockInfo(block)

			// check LastGoodBlock
			if blockNumber < lastGoodBlock {
				t.Fatalf("LastGoodBlock is wrong: ReorgedBlocks[%d] blockNumber=%v, LastGoodBlock=%v", i, blockNumber, lastGoodBlock)
			}
		}

		// check GoodBlocks
		for i, block := range result.GoodBlocks {
			blockNumber := block.Number
			hash := block.Hash

			if b, exists := tracker.getTrackerBlockInfo(blockNumber); exists {
				if b.Hash != hash {
					t.Fatalf("GoodBlocks[%d] is reorg: hash(before)=%v hash(after)=%v", i, b.Hash, hash)
				}
			}
			tracker.addTrackerBlockInfo(block)
		}

		syncChan <- struct{}{}
	}
}

func testEmitterCase3(t *testing.T) {
	tc := testCases[2]

	syncChan := make(chan struct{})
	filterResultChan := make(chan *superwatcher.FilterResult)
	errChan := make(chan error)

	conf, _ := config.ConfigYAML("../../config/config.yaml")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fakeDataGateway := &mockStateDataGateway{value: tc.fromBlock}
	sim := reorgsim.NewReorgSim(5, tc.fromBlock-1, tc.reorgedAt, tc.logs)
	e := New(conf, sim, fakeDataGateway, nil, nil, syncChan, filterResultChan, errChan, true)

	go func() {
		e.(*emitter).loopFilterLogs(ctx)
	}()

	tracker := newTracker()

	for {
		result := <-filterResultChan
		lastGoodBlock := result.LastGoodBlock

		// check ReorgedBlocks
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
			tracker.addTrackerBlockInfo(block)

			// check LastGoodBlock
			if blockNumber < lastGoodBlock {
				t.Fatalf("LastGoodBlock is wrong: ReorgedBlocks[%d] blockNumber=%v, LastGoodBlock=%v", i, blockNumber, lastGoodBlock)
			}
		}

		// check GoodBlocks
		for i, block := range result.GoodBlocks {
			blockNumber := block.Number
			hash := block.Hash

			if b, exists := tracker.getTrackerBlockInfo(blockNumber); exists {
				if b.Hash != hash {
					t.Fatalf("GoodBlocks[%d] is reorg: hash(before)=%v hash(after)=%v", i, b.Hash, hash)
				}
			}
			tracker.addTrackerBlockInfo(block)
		}

		syncChan <- struct{}{}
	}
}

func testEmitterCase4(t *testing.T) {
	tc := testCases[3]

	syncChan := make(chan struct{})
	filterResultChan := make(chan *superwatcher.FilterResult)
	errChan := make(chan error)

	conf, _ := config.ConfigYAML("../../config/config.yaml")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fakeDataGateway := &mockStateDataGateway{value: tc.fromBlock}
	sim := reorgsim.NewReorgSim(5, tc.fromBlock-1, tc.reorgedAt, tc.logs)
	e := New(conf, sim, fakeDataGateway, nil, nil, syncChan, filterResultChan, errChan, true)

	go func() {
		e.(*emitter).loopFilterLogs(ctx)
	}()

	tracker := newTracker()

	for {
		result := <-filterResultChan
		lastGoodBlock := result.LastGoodBlock

		// check ReorgedBlocks
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
			tracker.addTrackerBlockInfo(block)

			// check LastGoodBlock
			if blockNumber < lastGoodBlock {
				t.Fatalf("LastGoodBlock is wrong: ReorgedBlocks[%d] blockNumber=%v, LastGoodBlock=%v", i, blockNumber, lastGoodBlock)
			}
		}

		// check GoodBlocks
		for i, block := range result.GoodBlocks {
			blockNumber := block.Number
			hash := block.Hash

			if b, exists := tracker.getTrackerBlockInfo(blockNumber); exists {
				if b.Hash != hash {
					t.Fatalf("GoodBlocks[%d] is reorg: hash(before)=%v hash(after)=%v", i, b.Hash, hash)
				}
			}
			tracker.addTrackerBlockInfo(block)
		}

		syncChan <- struct{}{}
	}
}

func testEmitterCase5(t *testing.T) {
	tc := testCases[4]

	syncChan := make(chan struct{})
	filterResultChan := make(chan *superwatcher.FilterResult)
	errChan := make(chan error)

	conf, _ := config.ConfigYAML("../../config/config.yaml")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fakeDataGateway := &mockStateDataGateway{value: tc.fromBlock}
	sim := reorgsim.NewReorgSim(5, tc.fromBlock-1, tc.reorgedAt, tc.logs)
	e := New(conf, sim, fakeDataGateway, nil, nil, syncChan, filterResultChan, errChan, true)

	go func() {
		e.(*emitter).loopFilterLogs(ctx)
	}()

	tracker := newTracker()

	for {
		result := <-filterResultChan
		lastGoodBlock := result.LastGoodBlock

		// check ReorgedBlocks
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
			tracker.addTrackerBlockInfo(block)

			// check LastGoodBlock
			if blockNumber < lastGoodBlock {
				t.Fatalf("LastGoodBlock is wrong: ReorgedBlocks[%d] blockNumber=%v, LastGoodBlock=%v", i, blockNumber, lastGoodBlock)
			}
		}

		// check GoodBlocks
		for i, block := range result.GoodBlocks {
			blockNumber := block.Number
			hash := block.Hash

			if b, exists := tracker.getTrackerBlockInfo(blockNumber); exists {
				if b.Hash != hash {
					t.Fatalf("GoodBlocks[%d] is reorg: hash(before)=%v hash(after)=%v", i, b.Hash, hash)
				}
			}
			tracker.addTrackerBlockInfo(block)
		}

		syncChan <- struct{}{}
	}
}

type mockStateDataGateway struct {
	value uint64
}

func (m *mockStateDataGateway) GetLastRecordedBlock(context.Context) (uint64, error) {
	return m.value, nil
}

func (m *mockStateDataGateway) SetLastRecordedBlock(ctx context.Context, v uint64) error {
	m.value = v
	return nil
}

func (m *mockStateDataGateway) Shutdown() error {
	return nil
}
