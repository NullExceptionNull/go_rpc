package registry

import (
	"context"
	"encoding/json"
	"go.etcd.io/etcd/clientv3"
	"go_grpc/etcd"
	"log"
)

type ServiceInfo struct {
	Name string
	IP   string
}

type Service struct {
	ServiceInfo ServiceInfo
	stop        chan error
	leaseId     clientv3.LeaseID
	client      *clientv3.Client
	Name        string
}

//创建一个新的注册服务
func NewService(info ServiceInfo) (service *Service, err error) {
	return &Service{
		ServiceInfo: info,
		client:      etcd.Client,
		stop:        make(chan error),
	}, nil
}

func (s *Service) keepAlive() (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	info := s.ServiceInfo
	key := "services/" + s.Name
	value, _ := json.Marshal(info)

	resp, err := s.client.Grant(context.TODO(), 5)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	s.client.Put(context.TODO(), key, string(value), clientv3.WithLease(resp.ID))
}
