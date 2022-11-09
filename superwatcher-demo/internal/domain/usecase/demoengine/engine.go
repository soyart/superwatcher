package demoengine

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/artnoi43/superwatcher"

	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/domain/usecase/subengines"
)

type (
	// demoEngine wraps "subservices' engines"
	demoEngine struct {
		routes   map[subengines.SubEngineEnum]map[common.Address][]common.Hash
		services map[subengines.SubEngineEnum]superwatcher.ServiceEngine
	}
)

func New(
	routes map[subengines.SubEngineEnum]map[common.Address][]common.Hash,
	services map[subengines.SubEngineEnum]superwatcher.ServiceEngine,
) superwatcher.ServiceEngine {
	return &demoEngine{
		routes:   routes,
		services: services,
	}
}
