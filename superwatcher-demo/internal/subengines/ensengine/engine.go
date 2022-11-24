package ensengine

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-redis/redis/v8"

	"github.com/artnoi43/superwatcher"

	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/domain/entity"
	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/lib/contracts"
	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/lib/contracts/ens/enscontroller"
	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/lib/contracts/ens/ensregistrar"
	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/subengines"
)

const (
	eventNameRegistered string = "NameRegistered"
)

var ensEngineEvents = []string{eventNameRegistered}

type ensEngine struct {
	ensRegistrar  contracts.BasicContract
	ensController contracts.BasicContract
	dataGateway   entity.EnsDataGateway
}

type EnsSubEngineSuite struct {
	Engine      superwatcher.ServiceEngine // *ensEngine
	EnsRoutes   map[subengines.SubEngineEnum]map[common.Address][]common.Hash
	EnsServices map[subengines.SubEngineEnum]superwatcher.ServiceEngine
}

func New(registrarContract, controllerContract contracts.BasicContract, redisClient *redis.Client) superwatcher.ServiceEngine {
	return &ensEngine{
		ensRegistrar:  registrarContract,
		ensController: controllerContract,
		dataGateway:   *entity.NewEnsDataGateway(redisClient),
	}
}

// NewEnsSubEngineSuite returns a convenient struct for injecting into routerengine.routerEngine
func NewEnsSubEngineSuite() *EnsSubEngineSuite {
	registrarContract := contracts.NewBasicContract(
		"ENSRegistrar",
		ensregistrar.EnsRegistrarABI,
		"0x57f1887a8bf19b14fc0df6fd9b2acc9af147ea85",
		ensEngineEvents...,
	)
	controllerContract := contracts.NewBasicContract(
		"ENSController",
		enscontroller.EnsControllerABI,
		"0x283af0b28c62c092c9727f1ee09c02ca627eb7f5",
		ensEngineEvents...,
	)
	ensEngine := &ensEngine{
		ensRegistrar:  registrarContract,
		ensController: controllerContract,
	}

	registrarTopics := contracts.CollectEventHashes(registrarContract.ContractEvents)
	controllerTopics := contracts.CollectEventHashes(controllerContract.ContractEvents)

	return &EnsSubEngineSuite{
		Engine: ensEngine,
		EnsRoutes: map[subengines.SubEngineEnum]map[common.Address][]common.Hash{
			subengines.SubEngineENS: {
				registrarContract.Address:  registrarTopics,
				controllerContract.Address: controllerTopics,
			},
		},
		EnsServices: map[subengines.SubEngineEnum]superwatcher.ServiceEngine{
			subengines.SubEngineENS: ensEngine,
		},
	}
}

func NewEnsEngine(
	registrarContract contracts.BasicContract,
	controllerContract contracts.BasicContract,
) superwatcher.ServiceEngine {
	return &ensEngine{
		ensRegistrar:  registrarContract,
		ensController: controllerContract,
	}
}
