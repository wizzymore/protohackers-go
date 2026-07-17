package traffic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/reader"
	"github.com/wizzymore/tcp-go/server"
)

type connected struct{}
type disconnected struct{}

type message struct {
	client *server.TCPClient
	packet any
}

type TrafficServer struct {
	server *server.TCPServer

	ctx       context.Context
	close_ctx context.CancelFunc
	messages  chan message
	wg        sync.WaitGroup
}

func NewTrafficServer() (s server.Server, err error) {
	ts := &TrafficServer{}
	ts.server, err = server.NewTCPServer(ts.HandleClient)
	ts.messages = make(chan message, 32)
	ctx, close := context.WithCancel(context.Background())
	ts.ctx = ctx
	ts.close_ctx = close
	return ts, err
}

func (self *TrafficServer) Start() {
	self.wg.Add(1)
	go self.handlingServer()
	self.server.Start()
}

func (self *TrafficServer) Stop() error {
	self.close_ctx()
	self.wg.Wait()
	return self.server.Stop()
}

func (self *TrafficServer) HandleClient(client *server.TCPClient) (err error) {
	defer func(client *server.TCPClient) {
		self.messages <- message{
			client,
			disconnected{},
		}
	}(client)

	self.messages <- message{
		client,
		connected{},
	}

	for {
		var opcode byte
		opcode, err = reader.ReadByte(client.Conn)
		if err != nil {
			return
		}

		client.Logger.Debug().
			Hex("opcode", []byte{opcode}).
			Msg("received new opcode")

		var packet Packet

		switch opcode {
		case (*PlatePacket).Opcode(nil):
			packet = new(PlatePacket)
			err = packet.Unmarshal(client.Conn)
			if err != nil {
				return
			}
		case (*IAmCameraPacket).Opcode(nil):
			packet = new(IAmCameraPacket)
			err = packet.Unmarshal(client.Conn)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				return errors.Join(err, errors.New("could not unmarshal camera packet"))
			}
		case (*IAmDispatcherPacket).Opcode(nil):
			packet = new(IAmDispatcherPacket)
			err = packet.Unmarshal(client.Conn)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				return errors.Join(err, errors.New("could not unmarshal dispatcher packet"))
			}
		case (*WantHeartbeatPacket).Opcode(nil):
			packet = new(WantHeartbeatPacket)
			err = packet.Unmarshal(client.Conn)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				return errors.Join(err, errors.New("could not unmarshal want heartbeat packet"))
			}
		default:
			client.Logger.Warn().Hex("opcode", []byte{opcode}).Msg("received invalid opcode")
			errorPacket := ErrorPacket{"received invalid opcode"}
			var data []byte
			data, err = errorPacket.Marshal()
			if err != nil {
				return errors.Join(err, errors.New("could not marshal error packet"))
			}
			_, err = client.Write(data)
			if err != nil {
				return
			}
		}

		self.messages <- message{
			client,
			packet,
		}
	}
}

type PlateInfo struct {
	Plate     string
	Timestamp uint32
	Road      uint16
	Mile      uint16
}

type PeerId = uint
type Plate = string
type RoadId = uint16
type Day = int

func (self *TrafficServer) handlingServer() {
	defer self.wg.Done()

	clients := make(map[PeerId]*server.TCPClient)
	heartbeats := make(map[PeerId]context.CancelFunc)
	cameras := make(map[PeerId]*IAmCameraPacket)
	plateReadings := make(map[Plate][]*PlateInfo)
	dispatchers := make(map[PeerId]*IAmDispatcherPacket)
	road_dispatchers := make(map[uint16]*Balancer[PeerId])
	tickets := make(map[RoadId][]*TicketPacket)

	// This stores tags to plates ticketed on specific days so we don't ticket twice
	dayPlates := make(map[Day]map[Plate]struct{})

	for {
		select {
		case <-self.ctx.Done():
			log.Debug().Err(self.ctx.Err()).Msg("shutting down handlingServer")
			return
		case message := <-self.messages:
			switch packet := message.packet.(type) {
			case Packet:
				switch p := packet.(type) {
				case *WantHeartbeatPacket:
					{
						cancel, ok := heartbeats[message.client.Id]
						if ok {
							cancel()
							delete(heartbeats, message.client.Id)
						}
					}

					if p.Interval == 0 {
						continue
					}

					ctx, cancel := context.WithCancel(self.ctx)
					heartbeats[message.client.Id] = cancel
					self.wg.Add(1)
					go handleHeartbeath(message.client, p.Interval, ctx, &self.wg)
				case *IAmCameraPacket:
					if _, ok := cameras[message.client.Id]; ok {
						errorPacket := ErrorPacket{"you already are a camera"}
						data, _ := errorPacket.Marshal()
						_, _ = message.client.Write(data)
						break
					}
					if _, ok := dispatchers[message.client.Id]; ok {
						errorPacket := ErrorPacket{"you already are a dispatcher"}
						data, _ := errorPacket.Marshal()
						_, _ = message.client.Write(data)
						break
					}
					log.Info().Any("packet", p).Uint("peer", message.client.Id).Msg("a new camera connected")
					cameras[message.client.Id] = p
				case *IAmDispatcherPacket:
					if _, ok := dispatchers[message.client.Id]; ok {
						errorPacket := ErrorPacket{"you already are a dispatcher"}
						data, _ := errorPacket.Marshal()
						_, _ = message.client.Write(data)
						break
					}
					if _, ok := cameras[message.client.Id]; ok {
						errorPacket := ErrorPacket{"you already are a camera"}
						data, _ := errorPacket.Marshal()
						_, _ = message.client.Write(data)
						break
					}
					log.Info().Any("packet", p).Uint("peer", message.client.Id).Msg("a new dispatcher connected")
					dispatchers[message.client.Id] = p

					for _, road := range p.Roads {
						balancer := road_dispatchers[road]
						if balancer == nil {
							balancer = NewBalancer[PeerId]()
							road_dispatchers[road] = balancer
						}
						balancer.Add(message.client.Id)

						t, ok := tickets[road]
						if !ok || len(t) == 0 {
							continue
						}

						for _, ticket := range t {
							_ = sendTicket(message.client, ticket, -1)
						}

						tickets[road] = []*TicketPacket{}
					}
				case *PlatePacket:
					camera, ok := cameras[message.client.Id]
					if !ok {
						errorPacket := ErrorPacket{"you must be a camera to send plates"}
						data, _ := errorPacket.Marshal()
						_, _ = message.client.Write(data)
						break
					}

					newPacketDay := int(math.Floor(float64(p.Timestamp) / 86400))
					log.Info().Any("packet", p).
						Any("camera", camera).
						Uint("peer", message.client.Id).
						Int("day", newPacketDay).
						Msg("received a new plate")

					{
						// If we already send a ticket to this plate on this day, stop here
						// we can't ticket a plate twice in a specific day
						dp, ok := dayPlates[newPacketDay]
						if ok {
							if _, ok := dp[p.Plate]; ok {
								continue
							}
						}
					}

					plateInfo := &PlateInfo{
						Plate:     p.Plate,
						Timestamp: p.Timestamp,
						Road:      camera.Road,
						Mile:      camera.Mile,
					}

					plateReadings[p.Plate] = append(plateReadings[p.Plate], plateInfo)

					// Ticket checking
					sightings := plateReadings[p.Plate]

					// Not enough data to send a ticket
					if len(sightings) < 2 {
						continue
					}

					var current *PlateInfo
					var prev *PlateInfo
					var distance uint32
					var prevDay int
					var currentDay int

					for i := 0; i < len(sightings)-1; i++ {
						loopPacket := sightings[i]

						loopPacketDay := int(math.Floor(float64(loopPacket.Timestamp) / 86400))
						if loopPacket.Timestamp < plateInfo.Timestamp {
							current = plateInfo
							prev = loopPacket
							prevDay = loopPacketDay
							currentDay = newPacketDay
						} else {
							current = loopPacket
							prev = plateInfo
							prevDay = newPacketDay
							currentDay = loopPacketDay
						}

						{
							didReceiveTicket := false
							for i := prevDay; i <= currentDay; i++ {
								p, ok := dayPlates[i]
								if ok {
									if _, ok := p[current.Plate]; ok {
										didReceiveTicket = true
										break
									}
								}
							}

							// Skip this one we can't possibly send him a ticket on this day
							if didReceiveTicket {
								continue
							}
						}

						// Only check the same road
						if prev.Road != current.Road {
							continue
						}

						if prev.Mile < current.Mile {
							distance = uint32(current.Mile - prev.Mile)
						} else {
							distance = uint32(prev.Mile - current.Mile)
						}

						if distance <= 0 {
							continue
						}

						timeDiff := current.Timestamp - prev.Timestamp
						if timeDiff <= 0 {
							continue
						}

						speed := float32(distance*3600) / float32(timeDiff) // miles/hour

						if speed > float32(camera.Limit) {
							// ISSUE TICKET
							ticket := &TicketPacket{
								Plate:      current.Plate,
								Road:       camera.Road,
								Mile1:      prev.Mile,
								Timestamp1: prev.Timestamp,
								Mile2:      current.Mile,
								Timestamp2: current.Timestamp,
								Speed:      uint16(speed * 100),
							}

							for j := prevDay; j <= currentDay; j++ {
								log.Debug().
									Any("ticket", ticket).
									Int("day", j).
									Uint16("limit", camera.Limit).
									Float32("speed", speed).
									Msg("set ticket a sent for day")
								if dayPlates[j] == nil {
									dayPlates[j] = make(map[Plate]struct{})
								}
								dayPlates[j][current.Plate] = struct{}{}
							}

							rd, ok := road_dispatchers[current.Road]
							var dispatcher PeerId

							if ok {
								dispatcher, ok = rd.Get()
							}

							if !ok {
								// No dispatchers currently online for that road, queue it for later
								tickets[current.Road] = append(tickets[current.Road], ticket)
								log.Debug().
									Any("ticket", ticket).
									Int("day-start", prevDay).
									Int("day-end", currentDay).
									Uint16("limit", camera.Limit).
									Float32("speed", speed).
									Msg("no dispatcher available to send ticket, put in queue")
							} else {
								_ = sendTicket(clients[dispatcher], ticket, prevDay)
							}

							break
						}
					}

					// We must clean all the plate readings and remove all of them which are on days that received a ticket
					readings := plateReadings[p.Plate]
					for i := 0; i < len(readings); i++ {
						reading := readings[i]
						day := int(math.Floor(float64(reading.Timestamp) / 86400))
						if _, ok := dayPlates[day][reading.Plate]; ok {
							readings = append(readings[:i], readings[i+1:]...)
						}
					}
					plateReadings[p.Plate] = readings
				}
			case connected:
				clients[message.client.Id] = message.client
			case disconnected:
				delete(cameras, message.client.Id)
				if _, ok := dispatchers[message.client.Id]; ok {
					delete(dispatchers, message.client.Id)
					for road, dispatchers := range road_dispatchers {
						dispatchers.Remove(message.client.Id)
						if dispatchers.Len() == 0 {
							delete(road_dispatchers, road)
						}
					}
				}
				delete(heartbeats, message.client.Id)
			}
		}
	}
}

func sendTicket(client *server.TCPClient, ticket *TicketPacket, day int) error {
	data, err := ticket.Marshal()
	if err != nil {
		return err
	}
	_, err = client.Conn.Write(data)
	if err != nil {
		return err
	}
	log.Info().Any("ticket", ticket).Int("day", day).Uint("dispatcher", client.Id).Msg("sent ticket to dispatcher")

	return nil
}

func handleHeartbeath(client *server.TCPClient, interval uint32, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	d := time.Duration(float64(interval) / 10.0 * float64(time.Second))
	client.Logger.Info().
		Uint32("interval", interval).
		Dur("d", d).
		Msg("started new heartbeat handler")

	// interval is in deciseconds
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data, err := (*HeartbeatPacket).Marshal(nil)
			if err != nil {
				client.Logger.Err(err).Msg("could not marshal heartbeat packet")
				return
			}
			n, err := client.Write(data)
			if err != nil {
				if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
					return
				}
				client.Logger.Err(err).Msg("could not write heartbeat packet")
				return
			}
			if n != len(data) {
				client.Logger.Err(fmt.Errorf("expected to write %d but wrote only %d", len(data), n)).Msg("could not write full heartbeat packet to connection")
				return
			}

			// client.Logger.Info().Msg("sent heartbeat to client")
		}
	}
}
