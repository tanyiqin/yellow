package cluster

import (
	"net"
	"yellow/log"
	"yellow/parse"
)

type Session struct {
	// 连接套接字
	Conn net.Conn
	// cluster信息
	cluster *ServiceInfo
	// 编解码规则
	processor *parse.Processor
	// game service
	gameService *ServiceConn
}

func NewSession(conn net.Conn, cluster *ServiceInfo, processor *parse.Processor) *Session {
	s := &Session{
		Conn: conn,
		cluster: cluster,
		processor: processor,
		gameService: cluster.ProvideServiceConn(),
	}
	return s
}

func (s *Session) Run() {
	go s.startReader()
	go s.startWriter()
}

func (s *Session)startReader() {
	defer func() {
	}()
	for {
		data, err := s.processor.Read(s.Conn)
		if err != nil {
			log.Error("%+v", err)
			return
		}
		msg, err := s.processor.UnMarshal(data)
		if err != nil {
			log.Error("%+v", err)
		}

	}
}

func (s *Session)startWriter() {

}
