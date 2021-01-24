package cluster

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"net"
	"sync"
	"yellow/log"
)

type ServiceInfo struct {
	// 已连接的服务信息
	connectInfo map[string]net.Conn
	mutex sync.RWMutex
}

func NewServiceInfo() *ServiceInfo {
	s := &ServiceInfo{
		connectInfo: make(map[string]net.Conn),
	}
	return s
}

func (s *ServiceInfo)GetConnect(key string)net.Conn {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.connectInfo[key]
}

func (s *ServiceInfo)SetConnect(key string, val net.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.connectInfo[key] = val
}


// etcd心跳处理
func ListenAndDispose(keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse) {
	for _ = range keepAliveChan {
	}
	// 循环跳出意味着etcd心跳终止 需要处理
	log.Error("etcd keepalive wrong")
}

// 服务发现处理
func WatchDispose(ctx context.Context, s *ServiceInfo, watchChan <-chan clientv3.WatchResponse) {
	for wresp := range watchChan {
		for _, event := range wresp.Events {
			switch event.Type {
			case mvccpb.PUT:
				kv := event.Kv
				if s.GetConnect(string(kv.Key)) == nil {
					conn, err := net.Dial("tcp", string(kv.Key))
					if err != nil {
						log.Error("watch put dial err = ", err)
					} else {
						s.SetConnect(string(kv.Key), conn)
					}
				}
			case mvccpb.DELETE:
			}
		}
	}
}