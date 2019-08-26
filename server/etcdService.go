package server

import (
	"context"
	"fmt"
	etcdClient "go.etcd.io/etcd/client"
	"time"
)

type EtcdClient interface {
	RegisService(prefix, location string)
	GetServiceLocation(prefix string) ([]string, error)
}

type etcdClientImpl struct {
	connection etcdClient.KeysAPI
}

func NewEtcdClient(endpoints []string, timeOut int) EtcdClient {
	cfg := etcdClient.Config{
		Endpoints:               endpoints,
		Transport:               etcdClient.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	client, _ := etcdClient.New(cfg)
	connection := etcdClient.NewKeysAPI(client)
	return &etcdClientImpl{connection: connection}
}

func (e *etcdClientImpl) RegisService(prefix, location string) {
	key := fmt.Sprintf("%s/%s", prefix, location)
	for {
		_, err := e.connection.Set(context.Background(), key, "http://"+location, &etcdClient.SetOptions{
			TTL: time.Second * 10,
		})
		if err != nil {
			continue
		}
		time.Sleep(time.Second * 5)
	}
}

func (e *etcdClientImpl) GetServiceLocation(prefix string) (result []string, err error) {
	resp, err := e.connection.Get(context.Background(), fmt.Sprintf("%s/", prefix), &etcdClient.GetOptions{
		Recursive: true,
	})
	if resp.Node.Nodes.Len() >= 0 {
		for _, node := range resp.Node.Nodes {
			result = append(result, node.Value)
		}
	}

	return
}
