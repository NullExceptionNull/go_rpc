package registry

import (
	"go_grpc/common"
	"go_grpc/etcd"
	"sync"
	"testing"
)

func TestRegis(t *testing.T) {

	var wg sync.WaitGroup

	wg.Add(1)

	//初始化etcd
	etcd.Init()
	//
	node := common.Node{
		Name:    "activity",
		Ip:      "127.0.0.1",
		Port:    8080,
		Weight:  100,
		Healthy: true,
	}

	var registry = new(EtcdRegistry)

	registry.Init()

	registry.Register(&node)

	wg.Wait()

}
