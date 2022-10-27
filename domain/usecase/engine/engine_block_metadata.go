package engine

import (
	"fmt"
)

type blockMetadata struct {
	blockNumber uint64
	state       EngineBlockState
	artifacts   []Artifact
}

func (k blockMetadata) BlockNumber() uint64 {
	// TODO: Here for debugging
	if k.blockNumber == 0 {
		panic("got blockNumber 0 from a serviceLogStateKey")
	}
	return k.blockNumber
}

func (k blockMetadata) String() string {
	return fmt.Sprintf("%d", k.blockNumber)
}
