package jobcentre

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"slices"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/server"
)

type StatusType int

const (
	STATUS_OK StatusType = iota
	STATUS_ERROR
	STATUS_NO_JOB
)
const (
	STATUS_OK_MESSAGE     string = "ok"
	STATUS_ERROR_MESSAGE  string = "error"
	STATUS_NO_JOB_MESSAGE string = "no-job"
)

const UNKNOWN_ERROR = "unrecognized request type."

type JobData = map[string]any

type ErrorResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func WriteResponse(client *server.TCPClient, res any) error {
	data, err := json.Marshal(&res)

	if err != nil {
		return err
	}

	data = append(data, '\n')
	n, err := client.Write(data)
	if err != nil {
		return err
	}

	if len(data) != n {
		return io.ErrUnexpectedEOF
	}

	client.Logger.Debug().Any("res", res).Msg("sent new response to client")

	return nil
}

func WriteStatus(client *server.TCPClient, statusType StatusType, message string) error {
	var status string
	switch statusType {
	case STATUS_OK:
		status = STATUS_OK_MESSAGE
	case STATUS_ERROR:
		status = STATUS_ERROR_MESSAGE
	case STATUS_NO_JOB:
		status = STATUS_NO_JOB_MESSAGE
	}
	er := ErrorResponse{
		Status: status,
		Error:  message,
	}
	data, err := json.Marshal(&er)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	n, err := client.Write(data)
	if err != nil {
		return err
	}
	if len(data) != n {
		return io.ErrUnexpectedEOF
	}

	client.Logger.Debug().Any("res", er).Msg("sent new error message to client")

	return nil
}

type Request struct {
	Request string `json:"request"`
}

type GetRequest struct {
	Queues []string `json:"queues"`
	Wait   *bool    `json:"wait,omitempty"`
}

type GetResponse struct {
	Status string  `json:"status"`
	Id     int     `json:"id"`
	Job    JobData `json:"job"`
	Pri    uint    `json:"pri"`
	Queue  string  `json:"queue"`
}

type PutRequest struct {
	Queue string  `json:"queue"`
	Job   JobData `json:"job"`
	Pri   uint    `json:"pri"`
}

type PutResponse struct {
	Status string `json:"status"`
	Id     int    `json:"id"`
}

type AbortRequest struct {
	Id int `json:"id"`
}

type DeleteRequest struct {
	Id int `json:"id"`
}

type message struct {
	client  *server.TCPClient
	request any
}

type disconnected struct{}
type connected struct{}

type JobCentreServer struct {
	s          server.Server
	ctx        context.Context
	cancel_ctx context.CancelFunc
	messages   chan message
	wg         sync.WaitGroup
}

func NewJobCentreServer() (s server.Server, err error) {
	jc := &JobCentreServer{}
	jc.s, err = server.NewTCPServer(jc.handler)
	jc.ctx, jc.cancel_ctx = context.WithCancel(context.Background())
	jc.messages = make(chan message, 1024)
	s = jc
	if err != nil {
		return
	}

	return
}

func (self *JobCentreServer) Start() {
	self.wg.Add(1)
	go self.internal()
	self.s.Start()
}

func (self *JobCentreServer) Stop() error {
	self.cancel_ctx()
	self.wg.Wait()
	return self.s.Stop()
}

type PeerId = uint

type Job struct {
	Id       int
	data     JobData
	priority uint
	owner    uint
	Queue    string
}

type Waiter struct {
	ClientId uint
	Queues   []string
}

func (self *JobCentreServer) internal() error {
	defer self.wg.Done()

	jobNextId := 1

	clients := make(map[uint]*server.TCPClient)
	waiters := []Waiter{}

	queues := make(map[string][]*Job)
	jobs := []*Job{}

	for {
		select {
		case <-self.ctx.Done():
			log.Info().Msg("shutting down internal jobs handler")
			return self.ctx.Err()
		case message := <-self.messages:
			switch r := message.request.(type) {
			case connected:
				clients[message.client.Id] = message.client
			case disconnected:
				delete(clients, message.client.Id)
				waiters := slices.DeleteFunc(waiters, func(w Waiter) bool {
					return w.ClientId == message.client.Id
				})

				// If assigned jobs, put them back in their queue
				for _, job := range jobs {
					if job.owner == message.client.Id {
						job.owner = 0
						// If there is someone waiting on that queue, give it to them
						var foundWaiter bool
						for i, waiter := range waiters {
							if slices.Contains(waiter.Queues, job.Queue) {
								foundWaiter = true
								job.owner = waiter.ClientId
								waiters = append(waiters[:i], waiters[i+1:]...)
								res := GetResponse{
									Queue:  job.Queue,
									Job:    job.data,
									Pri:    job.priority,
									Status: STATUS_OK_MESSAGE,
									Id:     job.Id,
								}

								if err := WriteResponse(clients[waiter.ClientId], &res); err != nil {
									if !errors.Is(err, net.ErrClosed) {
										message.client.Logger.Err(err).Msg("could not write get response to waiter")
									}
								}
								log.Debug().Any("job", *job).Any("waiter", waiter).Msg("sent job to waiter - disconnected")
								break
							}
						}
						if !foundWaiter {
							log.Debug().Any("job", *job).Msg("put job back in queue - disconnected")
							queues[job.Queue] = append(queues[job.Queue], job)
						}
					}
				}
			case *PutRequest:
				job := &Job{
					Id:       jobNextId,
					data:     r.Job,
					priority: r.Pri,
					Queue:    r.Queue,
				}
				jobNextId += 1

				jobs = append(jobs, job)
				queues[r.Queue] = append(queues[r.Queue], job)

				res := PutResponse{
					Status: STATUS_OK_MESSAGE,
					Id:     job.Id,
				}

				if err := WriteResponse(message.client, &res); err != nil {
					if !errors.Is(err, net.ErrClosed) {
						message.client.Logger.Err(err).Msg("could not write put response to client")
					}
				}

				for i, waiter := range waiters {
					if slices.Contains(waiter.Queues, r.Queue) {
						waiters = append(waiters[:i], waiters[i+1:]...)
						res := GetResponse{
							Queue:  r.Queue,
							Job:    job.data,
							Pri:    r.Pri,
							Status: STATUS_OK_MESSAGE,
							Id:     job.Id,
						}

						if err := WriteResponse(clients[waiter.ClientId], &res); err != nil {
							if !errors.Is(err, net.ErrClosed) {
								message.client.Logger.Err(err).Msg("could not write get response to waiter - 2")
							}
						}
						break
					}
				}
			case *GetRequest:
				var job *Job
				for _, queueId := range r.Queues {
					q := queues[queueId]
					for _, j := range q {
						if job == nil || j.priority > job.priority {
							job = j
						}
					}
				}

				if job == nil {
					if r.Wait != nil && *r.Wait {
						waiters = append(waiters, Waiter{
							ClientId: message.client.Id,
							Queues:   r.Queues,
						})
						break
					}
					WriteStatus(message.client, STATUS_NO_JOB, "")
					break
				}

				res := GetResponse{
					Queue:  job.Queue,
					Job:    job.data,
					Pri:    job.priority,
					Status: STATUS_OK_MESSAGE,
					Id:     job.Id,
				}

				if err := WriteResponse(message.client, &res); err != nil {
					if !errors.Is(err, net.ErrClosed) {
						message.client.Logger.Err(err).Msg("could not write get response to client")
					}
				}

				// Remove the job from the queue
				job.owner = message.client.Id
				queues[job.Queue] = slices.DeleteFunc(queues[job.Queue], func(j *Job) bool {
					return j.Id == job.Id
				})
			case *AbortRequest:
				var job *Job
				for _, j := range jobs {
					if j.Id == r.Id {
						job = j
						break
					}
				}
				if job == nil || job.owner != message.client.Id {
					WriteStatus(message.client, STATUS_NO_JOB, "")
					break
				}

				queues[job.Queue] = append(queues[job.Queue], job)
				WriteStatus(message.client, STATUS_OK, "")
			case *DeleteRequest:
				var job *Job
				var idx int
				for i, j := range jobs {
					if j.Id == r.Id {
						idx = i
						job = j
						break
					}
				}
				if job == nil {
					WriteStatus(message.client, STATUS_NO_JOB, "")
					break
				}

				jobs = append(jobs[:idx], jobs[idx+1:]...)
				if job.owner == 0 {
					idx = slices.Index(queues[job.Queue], job)
					if idx != -1 {
						queues[job.Queue] = append(queues[job.Queue][:idx], queues[job.Queue][idx+1:]...)
					}
				}
				WriteStatus(message.client, STATUS_OK, "")
			}
		}
	}
}

func (self *JobCentreServer) handler(client *server.TCPClient) (err error) {
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

	reader := bufio.NewReader(client.Conn)
	var data []byte
	var request Request
	for {
		data, err = reader.ReadBytes('\n')
		if err != nil {
			return
		}
		err = json.Unmarshal(data, &request)
		if err != nil {
			if err = unknownRequest(client); err != nil {
				return
			}
		}

		var req any
		switch request.Request {
		case "put":
			req = new(PutRequest)
		case "get":
			req = new(GetRequest)
		case "abort":
			req = new(AbortRequest)
		case "delete":
			req = new(DeleteRequest)
		default:
			if err = unknownRequest(client); err != nil {
				return
			}
			continue
		}

		err = json.Unmarshal(data, req)
		if err != nil {
			if err = WriteStatus(client, STATUS_ERROR, "invalid put request"); err != nil {
				return
			}
		}
		client.Logger.Debug().
			Any("request", req).
			Type("request-type", req).
			Msg("received a new request")
		self.messages <- message{client, req}
	}
}

func unknownRequest(client *server.TCPClient) error {
	return WriteStatus(client, STATUS_ERROR, UNKNOWN_ERROR)
}

func noJob(client *server.TCPClient) error {
	return WriteStatus(client, STATUS_NO_JOB, UNKNOWN_ERROR)
}
