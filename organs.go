package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type WorkType int

func (w WorkType) String() string {
	switch w {
	case status:
		return "status"
	case cover:
		return "cover"
	case inhale:
		return "inhale"
	case exhale:
		return "exhale"
	case pump:
		return "pump"
	case move:
		return "move"
	case think:
		return "think"
	}
	return "unknown"
}

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

type StatusSocketData struct {
	Status      int              `json:"status"`
	Name        string           `json:"name"`
	Connections []string         `json:"connections"`
	WorkStatus  []WorkStatusData `json:"workStatus"`
}

type WorkStatusData struct {
	WorkType       string `json:"workType"`
	RequestCount   int    `json:"requestCount"`
	SuccessCount   int    `json:"successCount"`
	FailureCount   int    `json:"failureCount"`
	CompletedCount int    `json:"completedCount"`
}

type Edge struct {
	connection *websocket.Conn
}

func SendWork(connection *websocket.Conn, request Work) {
	err := websocket.JSON.Send(connection, WorkSocketData{
		WorkType: int(request.workType),
		Result:   request.result,
		Status:   request.status,
	})
	if err != nil {
		log.Fatal("Send: ", err)
	}
}

func SendStatus(connection *websocket.Conn, status StatusSocketData) error {
	return websocket.JSON.Send(connection, status)
}

func ReceiveWork(connection *websocket.Conn) (request Work) {
	data := &WorkSocketData{}
	err := websocket.JSON.Receive(connection, &data)
	if err != nil {
		log.Fatal("Receive: ", err)
	}
	request.workType = WorkType(data.WorkType)
	request.status = data.Status
	request.result = data.Result
	return
}

type Manager struct {
	sync.RWMutex
	nextAvailableWorker chan Worker
	resultChan          chan Work
	requestCount        int
	successCount        int
	failureCount        int
	completedCount      int
}

type Node struct {
	sync.RWMutex
	name         string
	edges        []*Edge
	serverMux    *http.ServeMux
	origin       string
	port         string
	websocketUrl string
	managers     map[WorkType]*Manager
	materialPool *MaterialPool
}

var currentPort = 7999

func GetNextAddresses() (string, string, string, string) {
	currentPort++
	return ORIGIN,
		fmt.Sprintf(":%v", currentPort),
		fmt.Sprintf(URL_TEMPLATE, currentPort),
		fmt.Sprintf(WEBSOCKET_URL_TEMPLATE, currentPort)
}

func InitializeNewNode(ctx context.Context, graph *Graph, name string) *Node {
	origin, port, url, websocketUrl := GetNextAddresses()
	node := &Node{
		name:         name,
		origin:       origin,
		port:         port,
		websocketUrl: websocketUrl,
		managers:     make(map[WorkType]*Manager),
	}
	node.materialPool = InitializeMaterialPool()
	graph.allNodes[url] = node
	node.Start(ctx)
	return node
}

func (n *Node) String() string {
	return fmt.Sprintf("%v%v (%T)", n.name, n.port, n)
}

func (n *Node) Start(ctx context.Context) {
	n.serverMux = http.NewServeMux()
	n.serverMux.Handle(WORK_ENDPOINT, websocket.Handler(n.ProcessIncomingWorkRequests))
	n.serverMux.Handle(STATUS_ENDPOINT, websocket.Handler(n.GetNodeStatus))

	go func() {
		err := http.ListenAndServe(n.port, n.serverMux)
		if err != nil {
			log.Fatal(fmt.Sprintf("%v ListenAndServe: ", n), err.Error())
		}
	}()
}

func (n *Node) Connect(ctx context.Context, origin, websocketUrl string) {
	connection, err := websocket.Dial(websocketUrl, "", origin)
	if err != nil {
		log.Fatal(fmt.Sprintf("%v Connect: ", n), err)
	}
	n.Lock()
	defer n.Unlock()
	n.edges = append(n.edges, &Edge{
		connection: connection,
	})
	go n.ProcessIncomingWorkResponses(ctx, connection)
}

func ConnectNodes(ctx context.Context, node1, node2 *Node) {
	node1.Connect(ctx, node2.origin, node2.websocketUrl)
	node2.Connect(ctx, node1.origin, node1.websocketUrl)
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
	worker.SetParent(nil)
}

func (n *Node) ProcessIncomingWorkRequests(connection *websocket.Conn) {
	defer connection.Close()
	for {
		work := ReceiveWork(connection)
		manager, ok := n.managers[work.workType]
		if ok {
			if work.status == 0 {
				// Recieved work request.
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
				select {
				case w := <-manager.nextAvailableWorker:
					finishedWork := w.Work(ctx, work)
					// fmt.Printf("%v Finished work: %v\n", n, finishedWork)
					SendWork(connection, finishedWork)
					manager.Lock()
					manager.completedCount++
					manager.Unlock()
				case <-ctx.Done():
				}
				cancel()
			} else {
				// Received completed work, which is not expected.
				log.Fatal(fmt.Sprintf("Should not receive completed work, got %v", work))
			}
		}
	}
}

func (n *Node) GetNodeStatus(connection *websocket.Conn) {
	defer connection.Close()
	ticker := time.NewTicker(5 * time.Second)
	for {
		<-ticker.C
		var connections []string
		for _, edge := range n.edges {
			address := strings.Replace(edge.connection.RemoteAddr().String(), "/work", "/status", 1)
			connections = append(connections, address)
		}
		var workStatus []WorkStatusData
		for workType, manager := range n.managers {
			manager.Lock()
			workStatus = append(workStatus, WorkStatusData{
				WorkType:       workType.String(),
				RequestCount:   manager.requestCount,
				SuccessCount:   manager.successCount,
				FailureCount:   manager.failureCount,
				CompletedCount: manager.completedCount,
			})
			manager.requestCount = 0
			manager.successCount = 0
			manager.failureCount = 0
			manager.completedCount = 0
			manager.Unlock()
		}
		err := SendStatus(connection, StatusSocketData{
			Status:      200,
			Name:        n.name,
			Connections: connections,
			WorkStatus:  workStatus,
		})
		if err != nil {
			return
		}
	}
}

func (n *Node) RequestWork(request Work) (result Work) {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT_SEC)
	defer cancel()
	// First try to get a result without sending:
	manager, ok := n.managers[request.workType]
	if !ok {
		manager = &Manager{
			nextAvailableWorker: make(chan Worker),
			resultChan:          make(chan Work, RESULT_BUFFER_SIZE),
		}
		n.managers[request.workType] = manager
	}
	manager.Lock()
	manager.requestCount++
	manager.Unlock()
	select {
	case <-ctx.Done():
		result = Work{
			workType: request.workType,
			result:   "Timeout",
			status:   503,
		}
	case result = <-manager.resultChan:
		// fmt.Printf("%v Received Result: %v\n", n, result)
	default:
		for _, edge := range n.edges {
			SendWork(edge.connection, request)
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
		case result = <-manager.resultChan:
			// fmt.Printf("%v Received Result: %v\n", n, result)
		}
	}
	manager.Lock()
	if result.status == 200 {
		manager.successCount++
	} else {
		manager.failureCount++
	}
	manager.Unlock()
	return
}

func (n *Node) ProcessIncomingWorkResponses(ctx context.Context, connection *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			work := ReceiveWork(connection)
			// fmt.Printf("%v Received Response: %v\n", n, work)
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
