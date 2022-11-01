package demoengine

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/artnoi43/superwatcher/pkg/logger"
	"github.com/artnoi43/superwatcher/pkg/superwatcher"
)

// MapLogToItem wraps mapLogToItem, so the latter can be unit tested.
func (e *demoEngine) HandleGoodLogs(
	logs []*types.Log,
	artifacts []superwatcher.Artifact,
) (
	[]superwatcher.Artifact,
	error,
) {
	logsMap := e.mapLogsToSubEngine(logs)
	var retArtifacts []superwatcher.Artifact // Artifacts to return

	for subEngine, logs := range logsMap {
		serviceEngine, ok := e.services[subEngine]
		if !ok {
			return nil, errors.Wrapf(errNoService, "subengine: %s", subEngine.String())
		}

		resultArtifacts, err := serviceEngine.HandleGoodLogs(logs, artifacts)
		if err != nil {
			return nil, errors.Wrapf(err, "subengine %s HandleGoodBlock failed", subEngine.String())
		}
		retArtifacts = append(retArtifacts, resultArtifacts...)
	}

	return retArtifacts, nil
}

func (e *demoEngine) HandleReorgedLogs(
	logs []*types.Log,
	artifacts []superwatcher.Artifact,

) ([]superwatcher.Artifact, error) {

	logsMap := e.mapLogsToSubEngine(logs)

	var retArtifacts []superwatcher.Artifact // Artifacts to return
	for subEngine, logs := range logsMap {
		serviceEngine, ok := e.services[subEngine]
		if !ok {
			return nil, errors.Wrapf(errNoService, "subengine", subEngine.String())
		}

		// Aggregate subEngine-specific artifacts
		var subEngineArtifacts []superwatcher.Artifact
		for _, artifact := range artifacts {
			if artifactIsFor(artifact, subEngine) {
				subEngineArtifacts = append(subEngineArtifacts, artifact)
			}
		}

		outputArtifacts, err := serviceEngine.HandleReorgedLogs(logs, subEngineArtifacts)
		if err != nil {
			return nil, errors.Wrapf(err, "subengine %s HandleReorgedBlock failed", subEngine.String())
		}

		retArtifacts = append(retArtifacts, outputArtifacts)
	}

	return retArtifacts, nil
}

// Unused by this service
func (e *demoEngine) HandleEmitterError(err error) error {
	logger.Warn("emitter error", zap.Error(err))

	return nil
}