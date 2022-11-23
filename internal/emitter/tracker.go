package emitter

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/wangjia184/sortedset"

	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/pkg/logger"
)

// TODO: blockInfoTracker name needs revision!
// TODO: Remove caller and logging
// blockInfoTracker stores the `superwatcher.BlockInfo` with blockNumber as key.
// It is used by emitter to store `BlockInfo` from the last run of `emitter.filterLogs`
// to see if a block's hash has changed.
type blockInfoTracker struct {
	sortedSet *sortedset.SortedSet
	user      string
}

func newTracker(user string) *blockInfoTracker {
	return &blockInfoTracker{
		sortedSet: sortedset.New(),
		user:      user,
	}
}

// addTrackerBlockInfo adds `*BlockInfo` |b| to the store using |b.Number| as key
func (t *blockInfoTracker) addTrackerBlockInfo(b *superwatcher.BlockInfo) {
	fmt.Println("emitter tracker: Adding block", t.user, b.Number, b.String(), "lenLogs", len(b.Logs))
	k := strconv.FormatInt(int64(b.Number), 10)
	t.sortedSet.AddOrUpdate(k, sortedset.SCORE(b.Number), b)
}

// getTrackerBlockInfo returns `*BlockInfo` from t with key |blockNumber|
func (t *blockInfoTracker) getTrackerBlockInfo(blockNumber uint64) (*superwatcher.BlockInfo, bool) {
	k := strconv.FormatUint(blockNumber, 10)
	node := t.sortedSet.GetByKey(k)
	if node == nil {
		return nil, false
	}
	val, ok := node.Value.(*superwatcher.BlockInfo)
	if !ok {
		logger.Panic(fmt.Sprintf("type assertion failed - expecting *BlockInfo, found %s", reflect.TypeOf(node.Value)))
	}
	return val, true
}

// clearUntil removes `*BlockInfo` in t from left to right.
func (t *blockInfoTracker) clearUntil(blockNumber uint64) {
	for {
		oldest := t.sortedSet.PeekMin()
		if oldest == nil || oldest.Score() > sortedset.SCORE(blockNumber) {
			break
		}

		t.sortedSet.PopMin()
	}
}

func (t *blockInfoTracker) Len() int {
	return t.sortedSet.GetCount()
}
