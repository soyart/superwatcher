package routerengine

import (
	"reflect"
	"testing"

	"github.com/soyart/gsl"

	"github.com/soyart/superwatcher"
	"github.com/soyart/superwatcher/pkg/reorgsim"

	"github.com/soyart/superwatcher/examples/demoservice/internal/domain/datagateway"
	"github.com/soyart/superwatcher/examples/demoservice/internal/subengines/ensengine"
	"github.com/soyart/superwatcher/examples/demoservice/internal/subengines/uniswapv3factoryengine"
)

func TestRouterArtifacts(t *testing.T) {
	logsPath := "../../../../testlogs/servicetest/logs_servicetest_16054000_16054100.json"
	logs := reorgsim.InitMappedLogsFromFiles(logsPath)

	var blocks []*superwatcher.Block
	for number, blockLogs := range logs {
		if len(blockLogs) == 0 {
			continue
		}

		blocks = append(blocks, &superwatcher.Block{
			Number: number,
			Logs:   gsl.CollectPointers(blockLogs),
		})
	}

	dgwENS := datagateway.NewMockDataGatewayENS()
	dgwPoolFactory := datagateway.NewMockDataGatewayPoolFactory()
	router := NewMockRouter(4, dgwENS, dgwPoolFactory)

	mapArtifacts, err := router.HandleGoodBlocks(blocks, nil)
	if err != nil {
		t.Error(err.Error())
	}

	if len(mapArtifacts) == 0 {
		t.Fatal("empty artifacts")
	}

	var artifacts []superwatcher.Artifact
	for _, outArtifacts := range mapArtifacts {
		artifacts = append(artifacts, outArtifacts...)
	}

	var artifactsENS []ensengine.ENSArtifact
	var artifactsPoolFactory []uniswapv3factoryengine.PoolFactoryArtifact
	for _, artifact := range artifacts {
		switch artifact.(type) {
		case ensengine.ENSArtifact:
			artifactsENS = append(artifactsENS, artifact.(ensengine.ENSArtifact))
		case uniswapv3factoryengine.PoolFactoryArtifact:
			artifactsPoolFactory = append(artifactsPoolFactory, artifact.(uniswapv3factoryengine.PoolFactoryArtifact))
		default:
			t.Fatalf("unexpected artifact type: %s", reflect.TypeOf(artifact).String())
		}
	}

	if len(artifactsENS) == 0 {
		t.Fatal("0 ENS artifacts")
	}
	if len(artifactsPoolFactory) == 0 {
		t.Fatal("0 PoolFactory artifacts")
	}
}
