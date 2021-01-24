package gate

import (
	"context"
	"fmt"
	"github.com/tanyiqin/lb"
	"net"
	"time"
	"yellow/cluster"
	"yellow/consts"
	"yellow/log"
	"yellow/session"
)

type gate struct {
	InnerAddr string
	OuterAddr string
	// etcd 地址
	EndPoints []string
	// gateID
	GateID int
	// 游戏服信息
	gameDiscovery *lb.ServiceDiscovery
	// 自身的服务注册
	myServiceRegister *lb.ServiceRegister
	// ctx
	ctx context.Context
	cancel context.CancelFunc
	cluster *cluster.ServiceInfo
}

func NewGate(gateID int, in, out string, endPoints []string) *gate {
	g := &gate{
		GateID: gateID,
		InnerAddr: in,
		OuterAddr: out,
		EndPoints: endPoints,
		cluster: cluster.NewServiceInfo(),
	}
	g.ctx, g.cancel = context.WithCancel(context.Background())
	return g
}

func (g *gate) GateName() string{
	return fmt.Sprintf("gate_service_%d", g.GateID)
}

func (g *gate) Run() {
	// 将自身的外部地址暴露 给login提供
	var err error
	g.myServiceRegister, err = lb.NewServiceRegister(g.EndPoints, "gates/" + g.GateName(), g.OuterAddr)
	if err != nil {
		log.Panic("err in register Service, err = ", err)
	}
	keepAliveChan, err := g.myServiceRegister.Run(g.ctx, int64(consts.GateEtcdKeepAliceLease))
	if err != nil {
		log.Panic("err in register Service, err = ", err)
	}
	go cluster.ListenAndDispose(keepAliveChan)

	// 通过服务发现找到后续模块 并与其连接
	g.gameDiscovery, err = lb.NewServiceDiscovery(g.EndPoints)
	if err != nil {
		log.Panic("err in discovery Service, err = ", err)
	}
	watchChan := g.gameDiscovery.Run(g.ctx, "games")
	go cluster.WatchDispose(g.ctx, g.cluster, watchChan)
	serverList := g.gameDiscovery.ServerList()
	for _, addr := range serverList {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Error("dial err = ", err)
		} else {
			g.cluster.SetConnect(addr, conn)
		}
	}

	// 开启loop循环 接受玩家的连接
	lis, err := net.Listen("tcp", g.OuterAddr)
	if err != nil {
		log.Panic("error in listen, err = ", err)
	}
	var timeDelay time.Duration
	for {
		conn, err := lis.Accept()
		if err != nil {
			if tempErr, ok := err.(*net.OpError); ok && tempErr.Temporary(){
				if timeDelay == 0 {
					timeDelay = 5 * time.Millisecond
				} else {
					timeDelay *= 2
				}
				if max := 1 * time.Second; timeDelay > max {
					timeDelay = max
				}
				timer := time.NewTimer(timeDelay)
				select {
					case <- timer.C:
					case <-g.ctx.Done():
						timer.Stop()
						return
				}
				continue
			}
			log.Error("bad accept, accept err = ", err)
		} else {
			timeDelay = 0
			sess := session.NewSession(conn, g.cluster)
			go sess.Run()
		}
	}
}