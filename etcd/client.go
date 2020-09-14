package etcd

import (
	"go.etcd.io/etcd/clientv3"
	"log"
	"sync"
	"time"
)

const ETCD_ADDR = ""

var once = sync.Once{}

var Client *clientv3.Client

func Init() {
	once.Do(InitEtcdClint)
}

func InitEtcdClint() {
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: []string{ETCD_ADDR},
		DialKeepAliveTimeout: 3 * time.Second})

	if err != nil {
		log.Fatal(err)
		return
	}
	Client = clientV3
}
