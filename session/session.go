package session

import (
	"net"
	"yellow/cluster"
)

type Session struct {
	// 连接套接字
	Conn net.Conn
	// cluster信息
	cluster *cluster.ServiceInfo
}

func NewSession(conn net.Conn, cluster *cluster.ServiceInfo) *Session {
	s := &Session{
		Conn: conn,
		cluster: cluster,
	}
	return s
}

func (s *Session) Run() {

}

func (s *Session)startReader() {
	for {

	}
}

func startWriter(conn net.Conn) {

}
