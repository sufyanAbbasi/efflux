package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type WorkType int

const (
	diffusion WorkType = iota
	cover              // Called on skin cells by muscle cells. Will randomly fail, i.e. cuts.
	exchange           // Called on blood cells by other cells.
	exhale             // Called on lung cells by blood cells.
	pump               // Called on to heart cells to pump, by brain cels.
	move               // Called on muscle cells by brain cells.
	think              // Called on brain cells to perform a computation, by muscle cells.
	digest             // Called on gut cells, by muscle cells.
	filter             // Called on kidney cellzs, by blood cells.
)

func (w WorkType) String() string {
	switch w {
	case cover:
		return "cover"
	case diffusion:
		return "diffusion"
	case digest:
		return "digest"
	case exhale:
		return "exhale"
	case filter:
		return "filter"
	case exchange:
		return "exchange"
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
	WorkType() WorkType
	SetOrgan(organ *Node)
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

type DiffusionSocketData struct {
	Resources ResourceBlobData
	Waste     WasteBlobData
}

type StatusSocketData struct {
	Status         int                `json:"status"`
	Name           string             `json:"name"`
	Connections    []string           `json:"connections"`
	WorkStatus     []WorkStatusData   `json:"workStatus"`
	MaterialStatus MaterialStatusData `json:"materialStatus"`
}

type WorkStatusData struct {
	WorkType              string `json:"workType"`
	RequestCount          int    `json:"requestCount"`
	SuccessCount          int    `json:"successCount"`
	FailureCount          int    `json:"failureCount"`
	CompletedCount        int    `json:"completedCount"`
	CompletedFailureCount int    `json:"completedFailureCount"`
}

type MaterialStatusData struct {
	O2         int `json:"o2"`
	Glucose    int `json:"glucose"`
	Vitamin    int `json:"vitamin"`
	Co2        int `json:"co2"`
	Creatinine int `json:"creatinine"`
	Growth     int `json:"growth"`
	Hunger     int `json:"hunger"`
	Asphyxia   int `json:"asphyxia"`
}

type TransportRequest struct {
	Name     string
	Base     []byte
	DNAType  int // Curve number, like elliptic.P384()
	CellType CellType
	WorkType WorkType
}

type Edge struct {
	workConnection *websocket.Conn
	transportUrl   string
}

func (n *Node) HandleTransportRequest(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields() // catch unwanted fields

	request := TransportRequest{}

	err := d.Decode(&request)
	if err != nil {
		// bad JSON or unrecognized json field
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cell, err := MakeCellFromRequest(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cell.SetOrgan(n)
	fmt.Println("Initialized:", cell, "in", cell.Organ())
	cell.Start(context.Background())

	fmt.Fprintf(w, "Success: Created %s", cell)
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

type WorkManager struct {
	sync.RWMutex
	nextAvailableWorker   chan Worker
	resultChan            chan Work
	requestCount          int
	successCount          int
	failureCount          int
	completedCount        int
	completedFailureCount int
}

type Node struct {
	sync.RWMutex
	ctx          context.Context
	name         string
	edges        []*Edge
	serverMux    *http.ServeMux
	origin       string
	port         string
	websocketUrl string
	transportUrl string
	managers     map[WorkType]*WorkManager
	materialPool *MaterialPool
	world        *World
}

var currentPort = 7999

func GetNextAddresses() (string, string, string, string, string) {
	currentPort++
	return ORIGIN,
		fmt.Sprintf(":%v", currentPort),
		fmt.Sprintf(URL_TEMPLATE, currentPort),
		fmt.Sprintf(WORK_URL_TEMPLATE, currentPort),
		fmt.Sprintf(TRANSPORT_URL_TEMPLATE, currentPort)
}

func InitializeNewNode(ctx context.Context, graph *Graph, name string) *Node {
	origin, port, url, websocketUrl, transportUrl := GetNextAddresses()
	node := &Node{
		ctx:          ctx,
		name:         name,
		origin:       origin,
		port:         port,
		websocketUrl: websocketUrl,
		transportUrl: transportUrl,
		managers:     make(map[WorkType]*WorkManager),
		world: &World{
			ctx: ctx,
			bounds: &image.Rectangle{
				Min: image.Point{
					X: -WORLD_BOUNDS,
					Y: -WORLD_BOUNDS,
				},
				Max: image.Point{
					X: WORLD_BOUNDS,
					Y: WORLD_BOUNDS,
				},
			},
			streamingChan: make(chan chan RenderableData),
			rootMatrix: &ExtracellularMatrix{
				RWMutex:  sync.RWMutex{},
				attached: make(map[RenderID]*Renderable),
			},
		},
	}
	node.world.rootMatrix.world = node.world
	node.materialPool = InitializeMaterialPool()
	graph.allNodes[url] = node
	node.Start(ctx)
	go node.world.Start(ctx)
	return node
}

func (n *Node) String() string {
	return fmt.Sprintf("%v (%v%v)", n.name, n.origin[:len(n.origin)-1], n.port)
}

func (n *Node) Start(ctx context.Context) {
	n.serverMux = http.NewServeMux()
	n.serverMux.Handle(WORK_ENDPOINT, websocket.Handler(n.ProcessIncomingWorkRequests))
	n.serverMux.Handle(STATUS_ENDPOINT, websocket.Handler(n.GetNodeStatus))
	n.serverMux.HandleFunc(TRANSPORT_ENDPOINT, n.HandleTransportRequest)
	if n.world != nil {
		n.serverMux.Handle(WORLD_ENDPOINT, websocket.Handler(n.world.Stream))
	}

	go func() {
		err := http.ListenAndServe(n.port, n.serverMux)
		if err != nil {
			log.Fatal(fmt.Sprintf("%v ListenAndServe: ", n), err.Error())
		}
	}()

	go func() {
		ticker := time.NewTicker(DIFFUSION_SEC)
		for {
			select {
			case <-ticker.C:
				n.SendDiffusion()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (n *Node) Connect(ctx context.Context, origin, websocketUrl string, transportUrl string) {
	connection, err := websocket.Dial(websocketUrl, "", origin)
	if err != nil {
		log.Fatal(fmt.Sprintf("%v Connect: ", n), err)
	}
	n.Lock()
	defer n.Unlock()
	n.edges = append(n.edges, &Edge{
		workConnection: connection,
		transportUrl:   transportUrl,
	})
	go n.ProcessIncomingWorkResponses(ctx, connection)
}

func ConnectNodes(ctx context.Context, node1, node2 *Node) {
	node1.Connect(ctx, node2.origin, node2.websocketUrl, node2.transportUrl)
	node2.Connect(ctx, node1.origin, node1.websocketUrl, node1.transportUrl)
}

func (n *Node) MakeAvailable(worker Worker) {
	go func(worker Worker) {
		manager, ok := n.managers[worker.WorkType()]
		if ok {
			if manager.nextAvailableWorker == nil {
				manager.nextAvailableWorker = make(chan Worker)
			}
			manager.nextAvailableWorker <- worker
		}
	}(worker)
}

func (n *Node) AddWorker(worker Worker) {
	n.Lock()
	defer n.Unlock()
	manager, ok := n.managers[worker.WorkType()]
	if ok {
		if manager.nextAvailableWorker == nil {
			manager.nextAvailableWorker = make(chan Worker)
		}
	} else {
		manager := &WorkManager{
			nextAvailableWorker: make(chan Worker),
			resultChan:          make(chan Work, RESULT_BUFFER_SIZE),
		}
		n.managers[worker.WorkType()] = manager
	}
	worker.SetOrgan(n)
}

func (n *Node) RemoveWorker(worker Worker) {
	n.Lock()
	defer n.Unlock()
	worker.SetOrgan(nil)
}

func (n *Node) ProcessIncomingWorkRequests(connection *websocket.Conn) {
	defer connection.Close()
	for {
		select {
		case <-n.ctx.Done():
			return
		default:
			work := ReceiveWork(connection)
			if n.materialPool != nil && work.workType == diffusion {
				n.ReceiveDiffusion(work)
				continue
			}
			manager, ok := n.managers[work.workType]
			if ok {
				if manager.nextAvailableWorker == nil {
					// Ignore: a request we can't handle with the workers we have.
					continue
				}
				if work.status == 0 {
					// Recieved work request.
					ctx, cancel := context.WithTimeout(context.Background(), WAIT_FOR_WORKER_SEC)
					select {
					case w := <-manager.nextAvailableWorker:
						finishedWork := w.Work(ctx, work)
						// fmt.Printf("%v Finished work: %v\n", n, finishedWork)
						SendWork(connection, finishedWork)
						manager.Lock()
						manager.completedCount++
						if finishedWork.status != 200 {
							manager.completedFailureCount++
							// Need more workers to complete the job.
							ligand := n.materialPool.GetLigand()
							if ligand.growth < LIGAND_GROWTH_THRESHOLD {
								ligand.growth++
							}
						}
						manager.Unlock()
					case <-ctx.Done():
						// Didn't have enough workers to process, we need more cells.
						// Signal growth ligand.
						ligand := n.materialPool.GetLigand()
						if ligand.growth < LIGAND_GROWTH_THRESHOLD {
							ligand.growth++
						}
						n.materialPool.PutLigand(ligand)
					}
					cancel()
				} else {
					// Received completed work, which is not expected.
					log.Fatal(fmt.Sprintf("Should not receive completed work, got %v", work))
				}
			}
		}
	}
}

func (n *Node) ReceiveDiffusion(request Work) {
	data := &DiffusionSocketData{}
	json.Unmarshal([]byte(request.result), data)

	resource := n.materialPool.GetResource()
	defer n.materialPool.PutResource(resource)
	resource.Add(&ResourceBlob{
		o2:       data.Resources.O2,
		glucose:  data.Resources.Glucose,
		vitamins: data.Resources.Vitamins,
	})
	waste := n.materialPool.GetWaste()
	defer n.materialPool.PutWaste(waste)
	waste.Add(&WasteBlob{
		co2:        data.Waste.CO2,
		creatinine: data.Waste.Creatinine,
	})
}

func (n *Node) GetNodeStatus(connection *websocket.Conn) {
	defer connection.Close()
	ticker := time.NewTicker(STATUS_SOCKET_CLOCK_RATE)
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			var connections []string
			for _, edge := range n.edges {
				address := strings.Replace(edge.workConnection.RemoteAddr().String(), "/work", "", 1)
				connections = append(connections, address)
			}
			var workStatus []WorkStatusData
			for workType, manager := range n.managers {
				manager.Lock()
				workStatus = append(workStatus, WorkStatusData{
					WorkType:              workType.String(),
					RequestCount:          manager.requestCount,
					SuccessCount:          manager.successCount,
					FailureCount:          manager.failureCount,
					CompletedCount:        manager.completedCount,
					CompletedFailureCount: manager.completedFailureCount,
				})
				manager.requestCount = 0
				manager.successCount = 0
				manager.failureCount = 0
				manager.completedCount = 0
				manager.completedFailureCount = 0
				manager.Unlock()
			}
			materialStatus := MaterialStatusData{
				O2:         n.materialPool.resourcePool.resources.o2,
				Glucose:    n.materialPool.resourcePool.resources.glucose,
				Vitamin:    n.materialPool.resourcePool.resources.vitamins,
				Co2:        n.materialPool.wastePool.wastes.co2,
				Creatinine: n.materialPool.wastePool.wastes.creatinine,
				Growth:     n.materialPool.ligandPool.ligands.growth,
				Hunger:     n.materialPool.ligandPool.ligands.hunger,
				Asphyxia:   n.materialPool.ligandPool.ligands.asphyxia,
			}
			err := SendStatus(connection, StatusSocketData{
				Status:         200,
				Name:           n.name,
				Connections:    connections,
				WorkStatus:     workStatus,
				MaterialStatus: materialStatus,
			})
			if err != nil {
				return
			}
		}
	}
}

func (n *Node) RequestWork(request Work) (result Work) {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT_SEC)
	defer cancel()
	// First try to get a result without sending:
	manager, ok := n.managers[request.workType]
	if !ok {
		manager = &WorkManager{
			// Skip nextAvailableWorker, to indicate that we can't process
			// requests of this type.
			resultChan: make(chan Work, RESULT_BUFFER_SIZE),
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
			SendWork(edge.workConnection, request)
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

func (n *Node) SendDiffusion() {
	if n.materialPool != nil {
		if len(n.edges) == 0 {
			return
		}
		// Pick a random edge to diffuse to.
		edge := n.edges[rand.Intn(len(n.edges))]

		// Grab a resource and waste blob to diffuse. Can be empty.
		resource := n.materialPool.SplitResource()
		waste := n.materialPool.SplitWaste()
		diffusionData, err := json.Marshal(DiffusionSocketData{
			Resources: ResourceBlobData{
				O2:       resource.o2,
				Glucose:  resource.glucose,
				Vitamins: resource.vitamins,
			},
			Waste: WasteBlobData{
				CO2:        waste.co2,
				Creatinine: waste.creatinine,
			},
		})
		if err != nil {
			log.Fatal("Could not send diffusion data: ", err)
		}
		SendWork(edge.workConnection, Work{
			workType: diffusion,
			result:   string(diffusionData),
			status:   0,
		})
	}
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
