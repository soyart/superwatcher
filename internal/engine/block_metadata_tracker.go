package engine

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/wangjia184/sortedset"
	"go.uber.org/zap"

	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/pkg/logger"
	"github.com/artnoi43/superwatcher/pkg/logger/debug"
)

type MetadataTracker interface {
	ClearUntil(blockNumber uint64)
	SetBlockMetadata(*superwatcher.BlockInfo, *blockMetadata)
	GetBlockMetadata(*superwatcher.BlockInfo) *blockMetadata
	SetBlockState(*superwatcher.BlockInfo, EngineBlockState)
	GetBlockState(*superwatcher.BlockInfo) EngineBlockState
}

// metadataTracker is an in-memory store for keeping engine internal states.
// It is used to decide whether or not to pass the logs to service engine.
type metadataTracker struct {
	sync.RWMutex

	// Field `Tracker.sortedSet` maps txHash to blockMetadata.
	// The score is blockNumber. This allow us to use ClearUntil.
	sortedSet *sortedset.SortedSet
	debug     bool
}

func NewTracker(debug bool) *metadataTracker {
	return &metadataTracker{
		sortedSet: sortedset.New(),
		debug:     debug,
	}
}

// ClearUntil removes items in t from left to right.
// TODO: Currently broken
func (t *metadataTracker) ClearUntil(blockNumber uint64) {
	t.Lock()
	defer t.Unlock()

	debug.DebugMsg(t.debug, "clearing engine state tracker", zap.Uint64("until", blockNumber))

	for {
		oldest := t.sortedSet.PeekMin()
		if oldest == nil || oldest.Score() > sortedset.SCORE(blockNumber) {
			break
		}

		t.sortedSet.PopMin()
	}
}

func (t *metadataTracker) SetBlockMetadata(b *superwatcher.BlockInfo, metadata *blockMetadata) {
	t.Lock()
	defer t.Unlock()

	t.sortedSet.AddOrUpdate(b.String(), sortedset.SCORE(b.Number), metadata)
}

func (t *metadataTracker) GetBlockMetadata(b *superwatcher.BlockInfo) *blockMetadata {
	t.RLock()
	defer t.RUnlock()

	node := t.sortedSet.GetByKey(b.String())
	// Avoid panicking when assert type on nil value
	if node == nil {
		return &blockMetadata{
			blockNumber: b.Number,
			blockHash:   b.String(),
		}
	}

	metadata, ok := node.Value.(*blockMetadata)
	if !ok {
		logger.Panic(
			fmt.Sprintf("type assertion failed - expecting EngineLogState, found %s", reflect.TypeOf(node.Value)),
		)
	}

	return metadata
}

func (t *metadataTracker) SetBlockState(b *superwatcher.BlockInfo, state EngineBlockState) {
	t.Lock()
	defer t.Unlock()

	metadata := t.GetBlockMetadata(b)
	if metadata == nil {
		// Create new metadata if null
		metadata = &blockMetadata{}
	}

	// Overwrite metadata.state
	metadata.state = state
	t.SetBlockMetadata(b, metadata)
}

func (t *metadataTracker) GetBlockState(b *superwatcher.BlockInfo) EngineBlockState {
	t.RLock()
	defer t.RUnlock()

	metadata := t.GetBlockMetadata(b)
	if metadata == nil {
		return EngineBlockStateNull
	}

	return metadata.state
}

func (t *metadataTracker) Len() int {
	t.RLock()
	defer t.RUnlock()

	return t.sortedSet.GetCount()
}