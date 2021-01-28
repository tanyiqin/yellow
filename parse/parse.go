package parse

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/tanyiqin/packet"
	"reflect"
)

type Processor struct {
	*packet.MsgPacket
	littleEndian bool
	msgMap map[uint16]*msgInfo
	msgID map[reflect.Type]uint16
}

type msgInfo struct {
	msgType reflect.Type
}

type Op func(*Processor)

func NewProcessor(op ...Op) *Processor {
	p := &Processor{
		littleEndian: false,
		msgMap: make(map[uint16]*msgInfo),
		msgID: make(map[reflect.Type]uint16),
		MsgPacket: packet.NewPacket(),
	}
	for _, f := range op {
		f(p)
	}
	return p
}

func NewMsgInfo(msgType reflect.Type) *msgInfo {
	m := &msgInfo{
		msgType: msgType,
	}
	return m
}

func LittleEndian(b bool) Op{
	return func(processor *Processor) {
		processor.littleEndian = b
	}
}

// 注册协议
func (p *Processor) Register(msg proto.Message) error {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		return fmt.Errorf("register err, not ptr")
	}
	if _, ok := p.msgID[msgType]; ok {
		return fmt.Errorf("register err, dumplicat msg")
	}

	mi := NewMsgInfo(msgType)
	id := uint16(len(p.msgMap))
	p.msgMap[id] = mi
	p.msgID[msgType] = id
	return nil
}

func (p *Processor) Marshal(_sid uint32, msg interface{}) ([]byte, error) {
	msgType := reflect.TypeOf(msg)

	_id, ok := p.msgID[msgType]
	if !ok {
		return nil, fmt.Errorf("not registe msg = %v", msgType)
	}

	id := make([]byte, 2)
	if p.littleEndian {
		binary.LittleEndian.PutUint16(id, _id)
	} else {
		binary.BigEndian.PutUint16(id, _id)
	}

	sid := make([]byte, 4)
	if p.littleEndian {
		binary.LittleEndian.PutUint32(sid, _sid)
	} else {
		binary.BigEndian.PutUint32(sid, _sid)
	}

	data, err := proto.Marshal(msg.(proto.Message))
	if err != nil {
		return nil, err
	}
	return bytes.Join([][]byte{id, sid, data}, []byte("")), nil
}

func (p *Processor) UnMarshal(data []byte) (uint32, interface{}, error) {
	if len(data) < 6 {
		return 0, nil, fmt.Errorf("protobuf data too short")
	}

	// 消息id字段
	var id uint16
	if p.littleEndian {
		id = binary.LittleEndian.Uint16(data)
	} else {
		id = binary.BigEndian.Uint16(data)
	}
	// session id 字段
	var sid uint32
	if p.littleEndian {
		sid = binary.LittleEndian.Uint32(data[2:])
	} else {
		sid = binary.BigEndian.Uint32(data[2:])
	}

	msg := reflect.New(p.msgMap[id].msgType.Elem()).Interface()
	return sid, msg, proto.UnmarshalMerge(data[6:], msg.(proto.Message))
}