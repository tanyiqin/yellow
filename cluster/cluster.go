package cluster

import (
	"context"
	"github.com/tanyiqin/lb"
	"go.etcd.io/etcd/clientv3"
	"yellow/consts"
	"yellow/log"
)

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