package routerengine

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"

	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/pkg/logger/debugger"

	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/domain/datagateway"
	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/subengines"
	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/subengines/ensengine"
	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/subengines/uniswapv3factoryengine"
)

// routerEngine wraps "subservices' engines"
type routerEngine struct {
	Routes   map[subengines.SubEngineEnum]map[common.Address][]common.Hash `json:"routes"`
	Services map[subengines.SubEngineEnum]superwatcher.ServiceEngine       `json:"services"`

	debugger *debugger.Debugger
}

func (e *routerEngine) String() string {
	b, err := json.Marshal(e)
	if err != nil {
		panic("failed to marshal routerEngine")
	}

	return string(b)
}

func New(
	routes map[subengines.SubEngineEnum]map[common.Address][]common.Hash,
	services map[subengines.SubEngineEnum]superwatcher.ServiceEngine,
	logLevel uint8,
) superwatcher.ServiceEngine {
	return &routerEngine{
		Routes:   routes,
		Services: services,
		debugger: debugger.NewDebugger("routerEngine", logLevel),
	}
}

func NewMockRouter(
	logLevel uint8,
	dataGatewayENS datagateway.DataGatewayENS,
	dataGatewayPoolFactory datagateway.DataGatewayPoolFactory,
) superwatcher.ServiceEngine {

	testSuiteENS := ensengine.NewTestSuiteENS(dataGatewayENS, logLevel)
	testSuitePoolFactory := uniswapv3factoryengine.NewTestSuitePoolFactory(dataGatewayPoolFactory, logLevel)

	routes := make(map[subengines.SubEngineEnum]map[common.Address][]common.Hash)
	routes[subengines.SubEngineENS] = testSuiteENS.Routes[subengines.SubEngineENS]
	routes[subengines.SubEngineUniswapv3Factory] = testSuitePoolFactory.Routes[subengines.SubEngineUniswapv3Factory]

	services := make(map[subengines.SubEngineEnum]superwatcher.ServiceEngine)
	services[subengines.SubEngineENS] = testSuiteENS.Engine
	services[subengines.SubEngineUniswapv3Factory] = testSuitePoolFactory.Engine

	return New(routes, services, logLevel)
}
