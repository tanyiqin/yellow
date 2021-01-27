package gate

import (
	"context"
	"fmt"
	"net"
	"time"
	"yellow/cluster"
	"yellow/log"
)

type gate struct {
	InnerAddr string
	OuterAddr string
	// etcd 地址
	EndPoints []string
	// gateID
	GateID int
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
	// 向etcd注册 服务
	cluster.EtcdRegister(g.ctx, g.EndPoints, "gates/out" + g.GateName(), g.OuterAddr)
	cluster.EtcdRegister(g.ctx, g.EndPoints, "gates/in" + g.GateName(), g.InnerAddr)

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
			sess := cluster.NewSession(conn, g.cluster)
			go sess.Run()
		}
	}
}

