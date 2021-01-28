package net

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
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
	RemoteID uint32
	// 编解码
	Processor *parse.Processor
	// 统一通过Channel来接收发消息
	RecvChan chan []byte
	// ctx
	Ctx context.Context
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

func (s *Session) Send(data[]byte) error {
	if s.closeFlag {
		return errors.New("sess closed")
	}
	s.RecvChan <- data
	return nil
}

func (s *Session) ID() uint32{
	return s.id
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
		Ctx: ctx,
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

func (sm *SessMgr)RandomSessID() uint32{
	return rand.Uint32() % uint32(len(sm.SessSet))
}

func (sm *SessMgr)Stop() {
	defer sm.mutex.Unlock()
	sm.mutex.Lock()
	for _, v := range sm.SessSet {
		v.Stop()
	}
}