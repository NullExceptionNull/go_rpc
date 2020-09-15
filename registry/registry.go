package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"go.etcd.io/etcd/clientv3"
	"go_grpc/common"
	"go_grpc/etcd"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	MaxSynCheckInterval = time.Second * 30
	DeleteCheckInterval = time.Second * 10
	MaxServiceChanSize  = 1000
	LeaseTime           = 100
	FalseInterval       = 15
	DeleteInterval      = 30
	Contract            = "%"
	NodePrefix          = "/Instance-"
)

type Instance struct {
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

func (e *EtcdRegistry) Register(node *common.Node) {
	//直接将node 放入 chan 中 判断逻辑放到消费者处处理
	e.NodeChan <- node
}

func (e *EtcdRegistry) UnRegister(node *common.Node) {
	//下线逻辑
	e.deleteFromEtcd(node)

	e.deleteFromMemory(node)
}

func (e *EtcdRegistry) getAllServiceName(serviceName string) *common.Service {
	return e.AllServiceMap[serviceName]
}

func (e *EtcdRegistry) heartBeat(node *common.Node) {
	//首先看这个服务在不在 如果存在就续约 如果不存在就更新
	_, ok := e.AllServiceMap[node.Name]

	if !ok {
		e.Register(node)
		return
	}
	node.LastHeartBeat = time.Now().Unix()
	node.Healthy = true
	e.update(node)
}

func (e *EtcdRegistry) put(node *common.Node) {
	resp, err := e.Client.Grant(context.TODO(), LeaseTime)
	if err != nil {
		log.Fatal(err)
		return
	}
	newUUID, err := uuid.NewUUID()
	node.Id = newUUID.String()
	node.LeaseId = resp.ID
	node.Healthy = true
	key := NodePrefix + node.Name + Contract + node.Ip
	node.Key = key
	node.LastHeartBeat = time.Now().Unix()
	nodeJson, _ := json.Marshal(node)
	_, err = e.Client.Put(context.TODO(), key, string(nodeJson), clientv3.WithLease(resp.ID))

	if err != nil {
		log.Fatal("存入 key 失败 ")
	}
	log.Printf("%s 注册成功 ", node.Ip)
}

func (e *EtcdRegistry) update(node *common.Node) {

	nodeJson, _ := json.Marshal(node)

	liveResp, err := e.Client.TimeToLive(context.TODO(), node.LeaseId)

	ttl := liveResp.TTL

	resp, _ := e.Client.Grant(context.TODO(), ttl)

	node.LeaseId = resp.ID

	_, err = e.Client.Put(context.TODO(), node.Key, string(nodeJson), clientv3.WithLease(resp.ID))

	if err != nil {
		log.Fatal("存入 key 失败 ")
	}
	log.Printf("%s 更新成功 ", node.Ip)
}

//从ETCD中删除node 节点
func (e *EtcdRegistry) deleteFromEtcd(node *common.Node) {

	e.Client.Delete(context.TODO(), node.Key)
}

//从内存中删除节点
func (e *EtcdRegistry) deleteFromMemory(node *common.Node) {

	nodes := e.AllServiceMap[node.Name].Nodes

	var deleteIndex int
	for index, targetNode := range nodes {
		if targetNode.Key == node.Key {
			deleteIndex = index
		}
	}
	nodes = append(nodes[:deleteIndex], nodes[deleteIndex+1:]...)
}

//服务续约
func (e *EtcdRegistry) renew(node *common.Node) {
	//找到服务的leaseId
	e.Client.KeepAliveOnce(context.TODO(), node.LeaseId)
}

func (e *EtcdRegistry) checkLastHeartBeat(nodeName string) {
	//直接从etcd 里面取 然后同步到内存
	response, err := e.Client.Get(context.TODO(), nodeName, clientv3.WithPrefix())

	if err != nil {
		log.Fatal(err)
		return
	}
	for _, nodeJson := range response.Kvs {
		var node common.Node
		err := json.Unmarshal(nodeJson.Value, &node)

		if err != nil {
			return
		}

		//判断两次IP更新时间 如果两次 超过 FalseInterval 置为false
		if time.Now().Unix()-node.LastHeartBeat > FalseInterval && node.Healthy {
			node.Healthy = false
			//同步到etcd 里面
			e.update(&node)
		}
		if time.Now().Unix()-node.LastHeartBeat > DeleteInterval {
			//需要移出
			e.deleteFromEtcd(&node)
			e.deleteFromMemory(&node)
		}
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
	deleteTicker := time.NewTicker(DeleteCheckInterval)

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
			fmt.Println("-deleteTicker--")
			response, err := e.Client.Get(context.TODO(), NodePrefix, clientv3.WithPrefix())

			if err != nil {
				continue
			}
			for _, nodeByte := range response.Kvs {
				var node common.Node
				err := json.Unmarshal(nodeByte.Value, &node)
				if err != nil {
					log.Printf("反序列化失败 %s", string(nodeByte.Value))
				}
				name := strings.Split(node.Key, Contract)[0]
				go e.checkLastHeartBeat(name)
			}
		default:
			time.Sleep(500 * time.Millisecond)
		}
	}
}
