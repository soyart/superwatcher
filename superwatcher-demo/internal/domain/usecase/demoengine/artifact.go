package demoengine

import (
	"github.com/artnoi43/superwatcher/pkg/superwatcher"

	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/domain/usecase/subengines"
)

// Exported for debugging
type DemoArtifact struct {
	// PoolFactoryArtifact is a hash map - so it can have 1 artifact for multiple logs
	PoolFactoryArtifact subEngineArtifact
	// EnsArtifact is a struct, so it needs an array to represent multiple logs
	EnsArtifact []subEngineArtifact
}

type subEngineArtifact interface {
	ForSubEngine() subengines.SubEngineEnum
}

func artifactIsFor(artifact superwatcher.Artifact, subEngine subengines.SubEngineEnum) bool {
	demoArtifact, ok := artifact.(subEngineArtifact)
	if !ok {
		return false
	}

	return demoArtifact.ForSubEngine() == subEngine
}