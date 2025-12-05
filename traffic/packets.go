//go:generate go run ../genpackets/genpackets.go -file packets.go

package traffic

type Camera struct {
	Road  uint16
	Mile  uint16
	limit uint16
}

type Dispatcher struct {
	Roads []uint16
}

type Error struct {
	Message string
}

type Plate struct {
	Plate     string
	Timestamp uint32
}

type Ticket struct {
	Plate      string
	Road       uint16
	Mile1      uint16
	Timestamp1 uint32
	Mile2      uint16
	Timestamp2 uint32
	Speed      uint16 // 100x Miles per hour
}

type WantHeartbeat struct {
	Interval uint32
}

type Heartbeat struct {
}
