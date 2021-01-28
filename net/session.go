package net

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"yellow/consts"
	"yellow/log"
	"yellow/parse"
)

var (
	sessionGlobalID uint32 = 1
)

type Session struct {
	net.Conn
	// 保管相同类型的session
	manager *SessMgr
	// 自身的ID
	id uint32
	// 连接的对端ID
	remoteID uint32
	// 编解码
	Processor *parse.Processor
	// 统一通过Channel来接收发消息
	RecvChan chan []byte
	// ctx
	ctx context.Context
	cancel context.CancelFunc
	// 是否关闭
	closeFlag bool
}

type SessMgr struct {
	// 保存了所有ID->Session的映射
	SessSet map[uint32]*Session
	// mutex
	mutex sync.RWMutex
}

func (s *Session) Run(tag int){
	go s.StartReader(tag)
	go s.StartWriter(tag)
	for {
		select {
			case <- s.ctx.Done():
		}
	}
}

func (s *Session)StartReader(tag int) {
	defer s.Stop()
	for {
		data, err := s.Processor.Read(s.Conn)
		if err != nil {
			log.Error("read msg error, err = ", err)
			break
		}
		_sid, msg, err := s.Processor.UnMarshal(data)
		if err != nil {
			log.Error("parse msg error, err = ", err)
			break
		}
		// 根据tag不同 不同处理
		switch tag {
			// 如果该连接 是由玩家连接到此的 需要根据session自身的id发送到对应的game中
			case consts.TagClient:
				// 如果remoteID = 0 说明刚进行连接 需要分配一个remoteID
				if s.remoteID == 0 {
					s.remoteID = 1
				}
				remoteSession =

		}
	}
}

func (s *Session) StartWriter(tag int) {
	defer s.Stop()
	for {
		select{
			case data := <- s.RecvChan:
				_, err := s.Conn.Write(data)
				if err != nil {
					log.Error("write msg err, err = ", err)
					return
				}
			case <- s.ctx.Done():
				return
		}
	}
}

func (s *Session)Stop() {
	if s.closeFlag {
		return
	}
	s.closeFlag = true
	s.cancel()
	s.manager.DelSession(s.id)
	s.Conn.Close()
}

func NewSessMgr() *SessMgr {
	sm := &SessMgr{
		SessSet: make(map[uint32]*Session),
	}
	return sm
}

func (sm *SessMgr)NewSession(conn net.Conn, processor *parse.Processor) (*Session, error){
	ctx, cancel := context.WithCancel(context.Background())
	s := &Session{
		Conn: conn,
		id: atomic.AddUint32(&sessionGlobalID, 1),
		Processor: processor,
		manager: sm,
		ctx: ctx,
		cancel: cancel,
		RecvChan: make(chan []byte),
		closeFlag:false,
	}
	err := sm.AddSession(s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (sm *SessMgr)AddSession(session *Session) error {
	defer sm.mutex.Unlock()
	sm.mutex.Lock()
	if _, ok := sm.SessSet[session.id]; ok {
		return fmt.Errorf("add sess error, dumplicate id = %d", session.id)
	}
	sm.SessSet[session.id] = session
	return nil
}

func (sm *SessMgr)DelSession(id uint32) {
	defer sm.mutex.Unlock()
	sm.mutex.Lock()
	delete(sm.SessSet, id)
}

func (sm *SessMgr)Stop() {
	defer sm.mutex.Unlock()
	sm.mutex.Lock()
	for _, v := range sm.SessSet {
		v.Stop()
	}
}