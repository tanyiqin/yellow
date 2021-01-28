package net

import (
	"context"
	"net"
	"time"
	"yellow/log"
	"yellow/parse"
)

type Server struct {
	ctx context.Context
	Addr string
	// 服务类型
	tag int
	// 连接管理
	sm *SessMgr
	// 编解码
	Processor *parse.Processor
	handler handleFunc
}

type handleFunc func(*Session)

func NewServer(ctx context.Context, Addr string, tag int, h handleFunc) *Server{
	s := &Server{
		ctx: ctx,
		Addr: Addr,
		tag: tag,
		sm: NewSessMgr(),
		handler: h,
	}
	return s
}

func (s *Server)Serve() {
	// 开启loop循环 接受连接
	lis, err := net.Listen("tcp", s.Addr)
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
				case <-s.ctx.Done():
					timer.Stop()
					return
				}
				continue
			}
			log.Error("bad accept, accept err = ", err)
		} else {
			timeDelay = 0
			sess, err := s.sm.NewSession(conn, s.Processor)
			if err != nil {
				conn.Close()
			} else {
				go s.handler(sess)
			}
		}
	}
}

func handlerClient(session *Session) {

}