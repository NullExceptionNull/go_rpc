package common

import "go.etcd.io/etcd/clientv3"

type Service struct {
	Name  string  `json:"name"`
	Nodes []*Node `json:"nodes"`
}

type Node struct {
	Name          string           `json:"name"` //这里的节点名称和service名称保持一致
	Id            clientv3.LeaseID `json:"id"`
	Ip            string           `json:"ip"`
	Port          int              `json:"port"`
	Weight        int              `json:"weight"`
	LastHeartBeat int64            `json:"last_heart_beat"`
}
