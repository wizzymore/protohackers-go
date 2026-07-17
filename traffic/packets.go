//go:generate go run ../genpackets/genpackets.go -file packets.go

package traffic

import "io"

type Packet interface {
	Marshal() (b []byte, err error)
	Unmarshal(r io.Reader) (err error)
	Opcode() byte
}

type IAmCameraPacket struct {
	Road  uint16
	Mile  uint16
	Limit uint16
}

func (self *IAmCameraPacket) Opcode() byte {
	return 0x80
}

type IAmDispatcherPacket struct {
	Roads []uint16
}

func (self *IAmDispatcherPacket) Opcode() byte {
	return 0x81
}

type ErrorPacket struct {
	Message string
}

func (self *ErrorPacket) Opcode() byte {
	return 0x10
}

type PlatePacket struct {
	Plate     string
	Timestamp uint32
}

func (self *PlatePacket) Opcode() byte {
	return 0x20
}

type TicketPacket struct {
	Plate      string
	Road       uint16
	Mile1      uint16
	Timestamp1 uint32
	Mile2      uint16
	Timestamp2 uint32
	Speed      uint16 // 100x Miles per hour
}

func (self *TicketPacket) Opcode() byte {
	return 0x21
}

type WantHeartbeatPacket struct {
	Interval uint32
}

func (self *WantHeartbeatPacket) Opcode() byte {
	return 0x40
}

type HeartbeatPacket struct {
}

func (self *HeartbeatPacket) Opcode() byte {
	return 0x41
}
