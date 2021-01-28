package gate

import (
	"context"
	"fmt"
	"yellow/cluster"
	"yellow/consts"
	"yellow/log"
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
	// server
	servers [2]*net.Server
}

func NewGate(gateID int, in, out string, endPoints []string) *gate {
	g := &gate{
		GateID: gateID,
		InnerAddr: in,
		OuterAddr: out,
		EndPoints: endPoints,
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

	g.servers[consts.TagClient] = net.NewServer(g.ctx, g.OuterAddr, consts.TagClient)
	g.servers[consts.TagServer] = net.NewServer(g.ctx, g.InnerAddr, consts.TagServer)

	for _, s := range g.servers {
		s.Serve(g.handleSession)
	}
}

func (g *gate) handleSession(session *net.Session, tag int) {
	go func() {
		for {
			select{
			case data := <- session.RecvChan:
				_, err := session.Conn.Write(data)
				if err != nil {
					log.Error("write msg err, err = ", err)
					return
				}
			case <- session.Ctx.Done():
				return
			}
		}
	}()
	for {
		data, err := session.Processor.Read(session.Conn)
		if err != nil {
			log.Error("read msg error, err = ", err)
			break
		}
		_sid, msg, err := session.Processor.UnMarshal(data)
		if err != nil {
			log.Error("parse msg error, err = ", err)
			break
		}
		// 根据tag不同 不同处理
		switch tag {
		// 如果该连接 是由玩家连接到此的 需要根据session自身的id发送到对应的game中
		case consts.TagClient:
			// 如果remoteID = 0 说明刚进行连接 需要分配一个remoteID
			if session.RemoteID == 0 {
				session.RemoteID = g.servers[consts.TagServer].Sm.RandomSessID()
			}
			// 将消息转发至对端
			RemoteSession, ok := g.servers[consts.TagServer].Sm.SessSet[session.RemoteID]
			if !ok {
				session.Stop()
				return
			}
			data, err = session.Processor.Marshal(session.ID(), msg)
			if err != nil {
				session.Stop()
				return
			}
			err = RemoteSession.Send(data)
			if err != nil {
				session.Stop()
				return
			}
			// 如果该连接由服务端连接到此 直接通过_sid来获取应该发往的端口
		case consts.TagServer:
			RemoteSession, ok := g.servers[consts.TagClient].Sm.SessSet[_sid]
			data, err = session.Processor.Marshal(session.ID(), msg)
			if !ok || err != nil {
				continue
			}
			RemoteSession.Send(data)
		}
	}
}

