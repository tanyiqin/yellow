package cluster

import (
	"context"
	"github.com/tanyiqin/lb"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"net"
	"net/rpc"
	"sync"
	"sync/atomic"
	"yellow/consts"
	"yellow/log"
)

type ServiceInfo struct {
	// 内部连接的服务信息
	connectInfo map[string]*ServiceTypeConn
	mutex sync.RWMutex
}

// 同一类型的conn
type ServiceTypeConn struct {
	prefix string
	name string
	conn *ServiceConn
}

// 本地route连接到其他服务的信息
type ServiceConn struct {
	*rpc.Client
	// 转发到此服务的用户数量
	num int64
}

func NewServiceInfo() *ServiceInfo {
	s := &ServiceInfo{
		connectInfo: make(map[string]*ServiceTypeConn),
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
	s.connectInfo[key] = &ServiceConn{
		Conn: val,
	}
}

// 指定分配一个ServiceConn
func(s *ServiceInfo) AssignServiceConn(key string) *ServiceConn{
	atomic.AddInt64(&s.connectInfo[key].num, 1)
	return s.connectInfo[key]
}

// 分配一个ServiceConn
func (s *ServiceInfo) ProvideServiceConn() *ServiceConn {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(s.connectInfo) <= 0 {
		return nil
	}
	var sc *ServiceConn
	for _, v := range s.connectInfo {
		if sc == nil || v.num <= sc.num{
			sc = v
		}
	}
	if sc != nil {
		sc.num++
	}
	return sc
}

func (s *ServiceInfo) GetNumber(key string) int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.connectInfo[key].num
}

func (s *ServiceConn) Run() {

}

// 注册服务
func EtcdRegister(ctx context.Context, endpoints []string, key, val string) {
	// 将自身的外部地址暴露 提供给客户端
	var err error
	ServiceRegister, err := lb.NewServiceRegister(endpoints, key, val)
	if err != nil {
		log.Panic("err in register Service, err = ", err)
	}
	keepAliveChan, err := ServiceRegister.Run(ctx, int64(consts.GateEtcdKeepAliceLease))
	if err != nil {
		log.Panic("err in register Service, err = ", err)
	}
	go ListenAndDispose(keepAliveChan)
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
			select {
				case <-ctx.Done():
					return
				default:
			}
		}
	}
}