package registry

import (
	"go.etcd.io/etcd/clientv3"
	"go_grpc/common"
	"go_grpc/etcd"
	"time"
)

const (
	MaxSynCheckInterval = time.Second * 30
	MaxServiceChanSize  = 1000
)

type RegistryInstance struct {
	Id      clientv3.LeaseID
	Service *common.Service
}

type EtcdRegistry struct {
	Client        *clientv3.Client
	NodeChan      chan *common.Node
	AllServiceMap map[string]*common.Service //类似于nacos 里面的注册表 key 是服务名称 value 是多个服务节点
}

func (e *EtcdRegistry) Init() {
	e.Client = etcd.Client
	e.NodeChan = make(chan *common.Node, MaxServiceChanSize)
	e.AllServiceMap = make(map[string]*common.Service, MaxServiceChanSize)
	go e.run()
}

func (e *EtcdRegistry) run() {
	ticker := time.NewTicker(MaxSynCheckInterval)

	for {
		select {
		case node := <-e.NodeChan: //每次进行
			//如果在这个chan 里面取出来了内容 则
			//首先判断注册表里面有没有 如果有就更新 如果没有就放到map里
			toRegisService, ok := e.AllServiceMap[node.Name]
			if ok {
				//需要更新一下心跳
				for _, oldNode := range toRegisService.Nodes {
					//判断是不是在这个node 里面 如果IP 一致的话 需要更新心跳
					if oldNode.Ip == node.Ip {
						oldNode.LastHeartBeat = time.Now().Unix()
					} else {
						//如果IP 不一致 这说明这是一个新的服务
						toRegisService.Nodes = append(toRegisService.Nodes, node)
					}
				}
				break
			}
			//如果注册表里面没有 需要初始化一个

		case <-ticker:
		default:

		}
	}
}
