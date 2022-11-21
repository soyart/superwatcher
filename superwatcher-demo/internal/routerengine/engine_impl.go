package routerengine

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/pkg/logger"
	"github.com/artnoi43/superwatcher/pkg/logger/debug"
	"github.com/artnoi43/superwatcher/superwatcher-demo/internal"
)

func (e *routerEngine) HandleGoodLogs(
	logs []*types.Log,
	artifacts []superwatcher.Artifact, // Ignored
) (
	[]superwatcher.Artifact,
	error,
) {
	// Artifacts to return - we don't know its size
	var retArtifacts []superwatcher.Artifact //nolint:prealloc

	logsMap := e.mapLogsToSubEngine(logs)
	for subEngine, logs := range logsMap {
		serviceEngine, ok := e.services[subEngine]
		if !ok {
			return nil, errors.Wrapf(errNoService, "subengine: %s", subEngine.String())
		}

		resultArtifacts, err := serviceEngine.HandleGoodLogs(logs, artifacts)
		if err != nil {
			if errors.Is(err, internal.ErrNoNeedHandle) {
				debug.DebugMsg(true, "routerEngine: got ErrNoNeedHandle", zap.String("subEngine", subEngine.String()))
				continue
			}
			return nil, errors.Wrapf(err, "subengine %s HandleGoodBlock failed", subEngine.String())
		}

		// Only append non-nil subengine artifacts
		if resultArtifacts != nil {
			// debug.DebugMsg(true, "got resultArtifacts", zap.Any("artifacts", resultArtifacts))
			retArtifacts = append(retArtifacts, resultArtifacts)
		}
	}

	return retArtifacts, nil
}

func (e *routerEngine) HandleReorgedLogs(
	logs []*types.Log,
	artifacts []superwatcher.Artifact,
) ([]superwatcher.Artifact, error) {
	logsMap := e.mapLogsToSubEngine(logs)

	var retArtifacts []superwatcher.Artifact //nolint:all Artifacts to return - we dont know the size
	for subEngine, logs := range logsMap {
		serviceEngine, ok := e.services[subEngine]
		if !ok {
			return nil, errors.Wrap(errNoService, "subengine: "+subEngine.String())
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
func (e *routerEngine) HandleEmitterError(err error) error {
	logger.Warn("emitter error", zap.Error(err))

	return nil
}
