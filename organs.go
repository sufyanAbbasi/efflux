package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

type WorkType int

type Worker interface {
	GetWorkType() WorkType
	SetParent(parent *Node)
	Work(ctx context.Context, request Work) Work
}

type Work struct {
	workType WorkType
	result   string
	status   int
}

type WorkSocketData struct {
	WorkType int
	Result   string
	Status   int
}

type Edge struct {
	connection *websocket.Conn
}

func Send(connection *websocket.Conn, request Work) {
	fmt.Printf("Send: %v\n", request)
	err := websocket.JSON.Send(connection, WorkSocketData{
		WorkType: int(request.workType),
		Result:   request.result,
		Status:   request.status,
	})
	if err != nil {
		log.Fatal("Send: ", err)
	}
}

func Receive(connection *websocket.Conn) (request Work) {
	data := &WorkSocketData{}
	err := websocket.JSON.Receive(connection, &data)
	if err != nil {
		log.Fatal("Receive: ", err)
	}
	request.workType = WorkType(data.WorkType)
	request.status = data.Status
	request.result = data.Result
	fmt.Printf("Receive: %v\n", request)
	return
}

type Manager struct {
	nextAvailableWorker chan Worker
	resultChan          chan Work
}

type Node struct {
	sync.RWMutex
	edges        []*Edge
	serverMux    *http.ServeMux
	origin       string
	port         string
	websocketUrl string
	managers     map[WorkType]*Manager
	workers      []Worker
}

func (n *Node) Start(ctx context.Context) {
	n.serverMux = http.NewServeMux()
	n.serverMux.Handle("/work", websocket.Handler(n.ProcessIncomingWorkRequests))

	go func() {
		err := http.ListenAndServe(n.port, n.serverMux)
		if err != nil {
			log.Fatal("ListenAndServe: " + err.Error())
		}
	}()
}

func (n *Node) Connect(ctx context.Context, origin, websocketUrl string) {
	ws, err := websocket.Dial(websocketUrl, "", origin)
	if err != nil {
		log.Fatal("Connect: ", err)
	}
	n.Lock()
	defer n.Unlock()
	n.edges = append(n.edges, &Edge{
		connection: ws,
	})
	go n.ProcessIncomingWorkResponses(ctx, ws)
}

func (n *Node) MakeAvailable(worker Worker) {
	go func(worker Worker) {
		manager, ok := n.managers[worker.GetWorkType()]
		if ok {
			manager.nextAvailableWorker <- worker
		}
	}(worker)
}

func (n *Node) AddWorker(worker Worker) {
	n.Lock()
	defer n.Unlock()
	n.workers = append(n.workers, worker)
	_, ok := n.managers[worker.GetWorkType()]
	if !ok {
		manager := &Manager{
			nextAvailableWorker: make(chan Worker),
			resultChan:          make(chan Work, RESULT_BUFFER_SIZE),
		}
		n.managers[worker.GetWorkType()] = manager
	}
	worker.SetParent(n)
}

func (n *Node) RemoveWorker(worker Worker) {
	n.Lock()
	defer n.Unlock()
	var workers []Worker
	for _, c := range n.workers {
		if c != worker {
			workers = append(workers, c)
		}
	}
	n.workers = workers
	worker.SetParent(nil)
}

func (n *Node) ProcessIncomingWorkRequests(connection *websocket.Conn) {
	defer connection.Close()
	for {
		fmt.Println("ProcessIncomingWorkRequests")
		work := Receive(connection)
		fmt.Printf("Received Request: %v\n", work)
		manager, ok := n.managers[work.workType]
		if ok {
			if work.status == 0 {
				// Recieved work request.
				ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT_SEC)
				select {
				case w := <-manager.nextAvailableWorker:
					finishedWork := w.Work(ctx, work)
					fmt.Printf("Finished work: %v\n", finishedWork)
					Send(connection, finishedWork)
				case <-ctx.Done():
				}
				cancel()
			} else {
				// Received completed work, which is not expected.
				log.Fatal(fmt.Sprintf("Should not receive completed work, got %v", work))
			}
		} else {
			fmt.Printf("Unable to fulfill request: %v\n", work)
		}
	}
}

func (n *Node) RequestWork(request Work) (result Work) {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT_SEC)
	defer cancel()
	// First try to get a result without sending:
	select {
	case <-ctx.Done():
		result = Work{
			workType: request.workType,
			result:   "Timeout",
			status:   503,
		}
	case result = <-n.managers[request.workType].resultChan:
		fmt.Printf("Received Result: %v\n", result)
	default:
		for _, edge := range n.edges {
			Send(edge.connection, request)
		}
	}
	if result.status == 0 {
		// No results were ready, try again, but block this time.
		select {
		case <-ctx.Done():
			result = Work{
				workType: request.workType,
				result:   "Timeout",
				status:   503,
			}
		case result = <-n.managers[request.workType].resultChan:
			fmt.Printf("Received Result: %v\n", result)
		}
	}
	return
}

func (n *Node) ProcessIncomingWorkResponses(ctx context.Context, connection *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Println("ProcessIncomingWorkResponses")
			work := Receive(connection)
			fmt.Printf("Received Response: %v\n", work)
			manager, ok := n.managers[work.workType]
			if ok {
				if work.status == 0 {
					// Received incompleted work, which is not expected.
					log.Fatal(fmt.Sprintf("Should not receive incompleted work, got %v", work))
				} else {
					// Received completed work.
					manager.resultChan <- work
				}
			} else {
				// Received completed work, but not expected.
				log.Fatal(fmt.Sprintf("Received completed work, but unable to process it. Got %v", work))
			}
		}
	}
}
