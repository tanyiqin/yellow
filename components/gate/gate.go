package gate

import (
	"context"
	"fmt"
	"yellow/cluster"
	"yellow/consts"
	"yellow/net"
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

	clientServer := net.NewServer(g.ctx, g.OuterAddr, consts.TagClient)
	go clientServer.Serve()
	serverServer := net.NewServer(g.ctx, g.InnerAddr, consts.TagServer)
	go serverServer.Serve()

}

