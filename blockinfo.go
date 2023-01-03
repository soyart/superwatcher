package superwatcher

import (
	"fmt"

	"github.com/artnoi43/gsl/gslutils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// BlockInfo represents the minimum block info needed for superwatcher.
type BlockInfo struct {
	// LogsMigrated indicates whether all interesting logs were moved/migrated
	// _from_ this block after a chain reorg or not. The field is primarily used
	// by EmitterPoller to trigger the poller to get new, fresh block Hash for a block.
	// The field should always be false if the BlockInfo is in PollResult.GoodBlocks.
	LogsMigrated bool `json:"logsMigrated"`

	Number uint64       `json:"number"`
	Hash   common.Hash  `json:"hash"`
	Header BlockHeader  `json:"-"`
	Logs   []*types.Log `json:"logs"`
}

// BlockHeader is implemented by `blockHeaderWrapper` and `*reorgsim.Block`.
// It is used in place of *types.Header to make writing tests with reorgsim easier.
// More methods may be added as our needs for data from the headers grow,
// or we (i.e. you) can mock the actual *types.Header in reorgsim instead :)
type BlockHeader interface {
	Hash() common.Hash
	Nonce() types.BlockNonce
	Time() uint64
	GasLimit() uint64
	GasUsed() uint64
}

// String returns the block hash with 0x prepended in all lowercase string.
func (b *BlockInfo) String() string {
	return gslutils.StringerToLowerString(b.Hash)
}

func (b *BlockInfo) BlockNumberString() string {
	return fmt.Sprintf("%d", b.Number)
}

type BlockHeaderWrapper struct {
	Header *types.Header
}

func (h *BlockHeaderWrapper) Hash() common.Hash {
	return h.Header.Hash()
}

func (h *BlockHeaderWrapper) Nonce() types.BlockNonce {
	return h.Header.Nonce
}

func (h *BlockHeaderWrapper) Time() uint64 {
	return h.Header.Time
}

func (h *BlockHeaderWrapper) GasLimit() uint64 {
	return h.Header.GasLimit
}

func (h *BlockHeaderWrapper) GasUsed() uint64 {
	return h.Header.GasUsed
}
