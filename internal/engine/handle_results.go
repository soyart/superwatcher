package engine

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (e *engine) handleResults(ctx context.Context) error {
	// Get emitterConfig to clear tracker metadata based on FilterRange
	emitterConfig := e.emitterClient.WatcherConfig()

	for {
		result := e.emitterClient.WatcherResult()
		if result == nil {
			e.debugger.Debug(1, "handleLogs got nil result, emitterClient was probably shutdown, returning..")
			return nil
		}

		var reorged bool
		for _, block := range result.ReorgedBlocks {
			reorged = true

			metadata := e.metadataTracker.GetBlockMetadata(block)
			metadata.state.Fire(EventReorg)

			// Only process block with Reorged state
			if metadata.state != StateReorged {
				e.debugger.Debug(
					1,
					"skip bad reorged block logs",
					zap.String("state", metadata.state.String()),
					zap.Uint64("blockNumber", metadata.blockNumber),
					zap.String("blockHash", metadata.blockHash),
				)

				continue
			}

			e.metadataTracker.SetBlockMetadata(block, metadata)

			artifacts, err := e.serviceEngine.HandleReorgedLogs(block.Logs, metadata.artifacts)
			if err != nil {
				return errors.Wrap(err, "serviceEngine.HandleReorgedBlockLogs failed")
			}

			// Check debug here so we dont have to iterate over all keys in map artifacts before checking in `e.debugMsg`
			if e.debug {
				for k, v := range artifacts {
					e.debugger.Debug(2, "got handleReorgedLogs artifacts", zap.Any("k", k), zap.Any("v", v))
				}
			}

			metadata.state.Fire(EventHandleReorg)
			metadata.artifacts = artifacts

			e.debugger.Debug(2, "saving handleReorgedLogs metadata for block", zap.Any("metadata", metadata))
			e.metadataTracker.SetBlockMetadata(block, metadata)
		}

		for _, block := range result.GoodBlocks {
			metadata := e.metadataTracker.GetBlockMetadata(block)
			metadata.state.Fire(EventGotLog)

			// Only process block with Seen state
			if metadata.state != StateSeen {
				e.debugger.Debug(
					1,
					"skip block",
					zap.String("state", metadata.state.String()),
					zap.Uint64("blockNumber", metadata.blockNumber),
					zap.String("blockHash", metadata.blockHash),
				)

				continue
			}

			artifacts, err := e.serviceEngine.HandleGoodLogs(block.Logs, metadata.artifacts)
			if err != nil {
				return errors.Wrap(err, "serviceEngine.HandleGoodBlockLogs failed")
			}

			metadata.state.Fire(EventProcess)
			metadata.artifacts = artifacts

			e.debugger.Debug(2, "saving metadata for block", zap.Any("metadata", metadata))
			e.metadataTracker.SetBlockMetadata(block, metadata)
		}

		// TODO: How many should we clear?
		e.metadataTracker.ClearUntil(
			result.LastGoodBlock - (emitterConfig.FilterRange * emitterConfig.GoBackRetries),
		)

		var lastRecordedBlock uint64
		if reorged {
			lastRecordedBlock = result.LastGoodBlock
		} else {
			lastRecordedBlock = result.ToBlock
		}

		if err := e.stateDataGateway.SetLastRecordedBlock(ctx, result.LastGoodBlock); err != nil {
			return errors.Wrapf(err, "failed to save lastRecordedBlock %d", lastRecordedBlock)
		}

		e.emitterClient.WatcherEmitterSync()
	}
}
