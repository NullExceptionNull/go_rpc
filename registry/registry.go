package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"go_grpc/common"
	"go_grpc/etcd"
	"log"
	"sync"
	"time"
)

const (
	MaxSynCheckInterval  = time.Second * 30
	MaxServiceChanSize   = 1000
	LeaseTime            = 15
	DeleteInterval       = time.Second * 15
	Contract             = "-"
	MaxHeartBeatInterval = time.Second * 30
)

type RegistryInstance struct {
	Id   clientv3.LeaseID
	Node *common.Node
}

type EtcdRegistry struct {
	Client        *clientv3.Client
	NodeChan      chan *common.Node
	AllServiceMap map[string]*common.Service //类似于nacos 里面的注册表 key 是服务名称 value 是多个服务节点
	lock          sync.Mutex
}

func (e *EtcdRegistry) Init() {
	e.Client = etcd.Client
	e.NodeChan = make(chan *common.Node, MaxServiceChanSize)
	e.AllServiceMap = make(map[string]*common.Service, MaxServiceChanSize)
	go e.run()
}

func (e *EtcdRegistry) Register(ctx context.Context, node *common.Node) {
	//首先看服务在不在 如果服务不在的话 需要创建一个新的服务
	//直接将node 放入 chan 中
	//申请一个10S 的租约
	e.NodeChan <- node
}

func (e *EtcdRegistry) put(node *common.Node) {
	resp, err := e.Client.Grant(context.TODO(), LeaseTime)
	if err != nil {
		log.Fatal(err)
		return
	}
	node.Id = resp.ID

	nodeJson, _ := json.Marshal(node)

	key := node.Name + Contract + node.Ip

	_, err = e.Client.Put(context.TODO(), key, string(nodeJson), clientv3.WithLease(resp.ID))
}

//服务续约
func (e *EtcdRegistry) renew(node *common.Node) {
	//找到服务的leaseId
	e.Client.KeepAliveOnce(context.TODO(), node.Id)
}

func (e *EtcdRegistry) checkLastHeartBeat(serviceName string) {
	//直接从etcd 里面取 然后同步到内存
	response, err := e.Client.Get(context.TODO(), serviceName+Contract, clientv3.WithPrefix())

	if err != nil {
		log.Fatal(err)
		return
	}
	for _, nodeJson := range response.Kvs {
		var node *common.Node
		err := json.Unmarshal(nodeJson.Value, node)
		if err != nil {
			return
		}
		//判断两次IP更新时间
		//if time.Now().Unix() - node.LastHeartBeat > MaxHeartBeatInterval {
		//}

	}

}

//服务踢出
func (e *EtcdRegistry) deleteNode(node *common.Node) {
	e.lock.Lock()
	var index int
	if service, ok := e.AllServiceMap[node.Name]; ok {
		for i, oleNode := range service.Nodes {
			if oleNode.Ip == node.Ip {
				index = i
			}
		}
	}
	nodes := e.AllServiceMap[node.Name].Nodes

	nodes = append(nodes[:index], nodes[index+1:]...)

	e.AllServiceMap[node.Name].Nodes = nodes

	e.lock.Unlock()
}

func (e *EtcdRegistry) run() {
	ticker := time.NewTicker(MaxSynCheckInterval)
	deleteTicker := time.NewTicker(DeleteInterval)

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
						e.renew(oldNode)
					} else {
						//如果IP 不一致 这说明这是一个新的服务
						toRegisService.Nodes = append(toRegisService.Nodes, node)
						e.put(node)
					}
				}
				break
			}
			//如果注册表里面没有 需要初始化一个 TODO 此处加锁
			newService := common.Service{Name: node.Name, Nodes: []*common.Node{node}}
			e.AllServiceMap[node.Name] = &newService
			e.put(node)

		case <-ticker.C:
			fmt.Println("---")

		case <-deleteTicker.C:
			//定期检查所有的node 上次更新时间

		default:
			time.Sleep(500 * time.Millisecond)
		}
	}
}
