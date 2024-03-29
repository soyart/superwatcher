package reorgsim

import (
	"fmt"
	"testing"

	"github.com/soyart/gsl"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

type multiReorgConfig struct {
	Name      string       `json:"name"`
	Param     Param        `json:"param"`
	Events    []ReorgEvent `json:"events"`
	LogsFiles []string     `json:"logsFiles"`
}

var (
	testsReorgSim = []multiReorgConfig{
		{
			LogsFiles: []string{
				logsPath + "/logs_lp.json",
				logsPath + "/logs_poolfactory.json",
			},
			Param: Param{
				StartBlock:    15944410,
				ExitBlock:     15944530,
				BlockProgress: DefaultParam.BlockProgress,
				Debug:         DefaultParam.Debug,
			},
			Events: []ReorgEvent{
				{
					ReorgBlock: 15944415,
					MovedLogs:  nil,
				},
			},
		},
		{
			LogsFiles: []string{
				logsPath + "/logs_lp.json",
				logsPath + "/logs_poolfactory.json",
			},
			Param: Param{
				StartBlock:    15944400,
				ExitBlock:     15944500,
				BlockProgress: DefaultParam.BlockProgress,
				Debug:         DefaultParam.Debug,
			},
			Events: []ReorgEvent{
				{
					ReorgBlock: 15944411,
					MovedLogs: map[uint64][]MoveLogs{
						15944411: {
							{
								NewBlock: 15944498,
								TxHashes: []common.Hash{
									common.HexToHash("0x1db603684cd6c04eec3166f216ebfb86c79bf63de6d0a9b2de535c38217d673d"),
								},
							},
						},
					},
				},
				{
					ReorgBlock: 15944455,
					MovedLogs: map[uint64][]MoveLogs{
						15944455: {
							{
								NewBlock: 15944498,
								TxHashes: []common.Hash{
									common.HexToHash("0x620be69b041f986127322985854d3bc785abe1dc9f4df49173409f15b7515164"),
								},
							},
						},
					},
				},
			},
		},
		{
			LogsFiles: []string{
				logsPath + "/logs_lp_5.json",
			},
			Param: Param{
				StartBlock:    15966490,
				ExitBlock:     15966540,
				BlockProgress: DefaultParam.BlockProgress,
				Debug:         DefaultParam.Debug,
			},
			Events: []ReorgEvent{
				{
					ReorgBlock: 15966522, // 0xf3a130
					// Move logs of 1 txHash to new block
					MovedLogs: map[uint64][]MoveLogs{
						15966522: { // 0xf3a13a
							{
								NewBlock: 15966527,
								TxHashes: []common.Hash{
									common.HexToHash("0x53f6b4200c700208fe7bb8cb806b0ce962a75e7a31d8a523fbc4affdc22ffc44"),
								},
							},
						},
					},
				},
				{
					ReorgBlock: 15966525, // 0xf3a130
					MovedLogs: map[uint64][]MoveLogs{ // 0xf3a13d
						15966525: {
							{
								NewBlock: 15966536, // 0xf3a13f
								TxHashes: []common.Hash{
									common.HexToHash("0xa46b7e3264f2c32789c4af8f58cb11293ac9a608fb335e9eb6f0fb08be370211"),
								},
							},
						},
					},
				},
			},
		},
	}
)

func TestReorgSim(t *testing.T) {
	for _, test := range testsReorgSim {
		err := testReorgSimMultiReorg(test)
		if err != nil {
			t.Error(err.Error())
		}
	}
}

func testReorgSimMultiReorg(testConf multiReorgConfig) error {
	sim, err := NewReorgSimFromLogsFiles(testConf.Param, testConf.Events, testConf.LogsFiles, testConf.Name, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create ReorgSim from config")
	}

	rSim := sim.(*ReorgSim)
	if len(rSim.reorgedChains) != len(testConf.Events) {
		return errors.New("len reorgedChain doesn't match len ReorgEvents")
	}

	for i, event := range testConf.Events {
		var prevChain BlockChain

		if i == 0 {
			prevChain = rSim.currentChain
		} else {
			prevChain = rSim.reorgedChains[i-1]
		}

		reorgedChain := rSim.reorgedChains[i]
		for blockNumber, b := range rSim.currentChain {
			reorgedBlock, ok := reorgedChain[blockNumber]
			if !ok {
				return fmt.Errorf("original block %d not found in reorgedChain[%d]", blockNumber, i)
			}

			if blockNumber >= event.ReorgBlock {
				if b.hash == reorgedBlock.hash {
					return fmt.Errorf("reorgedBlock %d has original hash %s", blockNumber, reorgedBlock.hash.String())
				}
			}
		}

		for blockMovedFrom, moves := range event.MovedLogs {
			for _, move := range moves {
				_, ok := prevChain[blockMovedFrom]
				if !ok {
					return fmt.Errorf("moveFromBlock %d not found in prevChain", blockMovedFrom)
				}

				movedFromBlock, ok := reorgedChain[blockMovedFrom]
				if !ok {
					return fmt.Errorf("moveFromBlock %d not found in reorgedChain[%d]", blockMovedFrom, i)
				}

				moveToBlock, ok := reorgedChain[move.NewBlock]
				if !ok {
					return fmt.Errorf("moveToBlock %d not found in reorgedChain[%d]", move.NewBlock, i)
				}

				// movedFromBlock should not have any logs with TxHash in move.TxHashes
				for _, log := range movedFromBlock.logs {
					if gsl.Contains(move.TxHashes, log.TxHash) {
						return fmt.Errorf("moveFromBlock still has log %s", log.TxHash.String())
					}
				}

				// Check if all move.TxHashes has actually been moved to move.NewBlock
				var count int
				var seen = make(map[common.Hash]bool)
				for _, log := range moveToBlock.logs {
					if seen[log.TxHash] {
						continue
					}

					seen[log.TxHash] = true

					if gsl.Contains(move.TxHashes, log.TxHash) {
						count++
					}
				}

				if l := len(move.TxHashes); l != count {
					return fmt.Errorf(
						"expecting %d logs to move from %d to %d, only got %d",
						l, blockMovedFrom, move.NewBlock, count,
					)
				}
			}
		}
	}

	return nil
}
