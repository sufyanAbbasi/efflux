package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WorkType int

const (
	nothing WorkType = iota
	diffusion
	cover    // Called on skin cells by muscle cells. Will randomly fail, i.e. cuts.
	exchange // Called on blood cells by other cells.
	exhale   // Called on lung cells by blood cells.
	pump     // Called on to heart cells to pump, by brain cels.
	move     // Called on muscle cells by brain cells.
	think    // Called on brain cells to perform a computation, by muscle cells.
	digest   // Called on gut cells, by muscle cells.
	filter   // Called on kidney cellzs, by blood cells.
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
	Hormone   HormoneBlobData
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
	O2           int `json:"o2"`
	Glucose      int `json:"glucose"`
	Vitamin      int `json:"vitamin"`
	Co2          int `json:"co2"`
	Creatinine   int `json:"creatinine"`
	Growth       int `json:"growth"`
	Hunger       int `json:"hunger"`
	Asphyxia     int `json:"asphyxia"`
	Inflammation int `json:"inflammation"`
	G_CSF        int `json:"g_csf"`
	M_CSF        int `json:"m_csf"`
}

type TransportRequest struct {
	Name           string
	Base           []byte
	DNAType        int // Curve number, like elliptic.P384()
	CellType       CellType
	WorkType       WorkType
	ParentRenderID string
	TransportPath  [10]string
	WantPath       [10]string
}

type EdgeType int

const (
	cardiovascular EdgeType = iota
	neuronal
	lymphatic
	muscular
	skeletal
	blood_brain_barrier
)

type Edge struct {
	edgeType       EdgeType
	workConnection *Connection
	transportUrl   string
}

func MakeTransportRequest(
	transportUrl string,
	name string,
	dna *DNA,
	cellType CellType,
	workType WorkType,
	parentRenderID string,
	transportPath [10]string,
	wantPath [10]string,
) error {
	dnaBase, err := dna.Serialize()
	if err != nil {
		log.Fatal("Transport: ", err)
	}
	dnaType := 0
	for i, d := range DNATypeMap {
		if d == dna.dnaType {
			dnaType = i
		}
	}
	jsonData, err := json.Marshal(TransportRequest{
		Name:           name,
		Base:           dnaBase,
		DNAType:        dnaType,
		CellType:       cellType,
		WorkType:       workType,
		ParentRenderID: parentRenderID,
		TransportPath:  transportPath,
		WantPath:       wantPath,
	})
	if err != nil {
		return fmt.Errorf("transport error: %v", err)
	}
	request, err := http.NewRequest("POST", transportUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("transport error: %v", err)
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("transport error: %v", err)
	}
	defer response.Body.Close()
	return nil
}

func (n *Node) HandleTransportRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields() // catch unwanted fields

	request := TransportRequest{}

	err := d.Decode(&request)
	if err != nil {
		// bad JSON or unrecognized json field
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cell, err := n.MakeCellFromRequest(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cell.SetOrgan(n)
	cell.RecordTransport()
	if n.verbose {
		fmt.Println("Spawned:", cell, "in", cell.Organ())
	}
	ctx, stop := context.WithCancel(ctx)
	cell.SetStop(stop)
	cell.Start(ctx)

	fmt.Fprintf(w, "Success: Created %s", cell)
}

func (n *Node) MakeCellFromRequest(request TransportRequest) (CellActor, error) {
	dna, err := MakeDNAFromRequest(request)
	if err != nil {
		return nil, err
	}
	var render *Renderable
	if request.ParentRenderID != "" {
		renderId := RenderID(request.ParentRenderID)
		if n.tissue != nil {
			render = n.tissue.FindRender(renderId)
		}
	}
	if render == nil {
		render = &Renderable{}
	}
	return MakeCellFromType(request.CellType, request.WorkType, dna, render, request.TransportPath, request.WantPath), nil
}

func SendWork(connection *Connection, request Work) {
	err := connection.WriteJSON(WorkSocketData{
		WorkType: int(request.workType),
		Result:   request.result,
		Status:   request.status,
	})
	if err != nil {
		log.Fatal("Send: ", err)
	}
}

func SendStatus(connection *Connection, status StatusSocketData) error {
	return connection.WriteJSON(status)
}

func ReceiveWork(connection *Connection) (request Work) {
	data := &WorkSocketData{}
	err := connection.ReadJSON(data)
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
	name         string
	edges        []*Edge
	serverMux    *http.ServeMux
	origin       string
	port         string
	websocketUrl string
	transportUrl string
	managers     *sync.Map
	materialPool *MaterialPool
	tissue       *Tissue
	verbose      bool
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

func InitializeNewNode(ctx context.Context, graph *Graph, name string, verbose bool) *Node {
	origin, port, url, websocketUrl, transportUrl := GetNextAddresses()
	node := &Node{
		name:         name,
		origin:       origin,
		port:         port,
		websocketUrl: websocketUrl,
		transportUrl: transportUrl,
		managers:     &sync.Map{},
		tissue:       InitializeTissue(ctx),
		verbose:      verbose,
	}
	node.materialPool = InitializeMaterialPool(ctx)
	graph.allNodes[url] = node
	node.Start(ctx)
	go node.tissue.Start(ctx)
	return node
}

func (n *Node) String() string {
	return fmt.Sprintf("%v (%v%v)", n.name, n.origin[:len(n.origin)-1], n.port)
}

type Connection struct {
	readMu  sync.RWMutex
	writeMu sync.RWMutex
	*websocket.Conn
}

func (c *Connection) WriteJSON(v interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.Conn.WriteJSON(v)
}

func (c *Connection) ReadJSON(v interface{}) error {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	return c.Conn.ReadJSON(v)
}

func (c *Connection) Close() error {
	return c.Conn.Close()
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WebsocketHandler(ctx context.Context, connectionHandler func(context.Context, *Connection)) func(w http.ResponseWriter, r *http.Request) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		connectionHandler(ctx, &Connection{
			Conn: conn,
		})
	}
	return handler
}

func (n *Node) Start(ctx context.Context) {
	n.serverMux = http.NewServeMux()
	n.serverMux.HandleFunc(WORK_ENDPOINT, WebsocketHandler(ctx, n.ProcessIncomingWorkRequests))
	n.serverMux.HandleFunc(STATUS_ENDPOINT, WebsocketHandler(ctx, n.GetNodeStatus))
	n.serverMux.HandleFunc(TRANSPORT_ENDPOINT, func(w http.ResponseWriter, r *http.Request) {
		n.HandleTransportRequest(ctx, w, r)
	})
	if n.tissue != nil {
		n.serverMux.HandleFunc(WORLD_ENDPOINT, WebsocketHandler(ctx, n.tissue.Stream))
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
				n.SendDiffusion(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (n *Node) Connect(ctx context.Context, origin, websocketUrl string, transportUrl string, edgeType EdgeType) error {
	dialer := websocket.Dialer{}
	connection, response, err := dialer.DialContext(ctx, websocketUrl, http.Header{})
	if err != nil || response.StatusCode != 101 {
		return fmt.Errorf("%v Connect: %w", n, err)
	}
	n.Lock()
	defer n.Unlock()
	conn := &Connection{
		Conn: connection,
	}
	n.edges = append(n.edges, &Edge{
		edgeType:       edgeType,
		workConnection: conn,
		transportUrl:   transportUrl,
	})
	go n.ProcessIncomingWorkResponses(ctx, conn)
	return nil
}

func ConnectNodes(ctx context.Context, node1, node2 *Node, toNode2 EdgeType, toNode1 EdgeType) {
	node1.Connect(ctx, node2.origin, node2.websocketUrl, node2.transportUrl, toNode2)
	node2.Connect(ctx, node1.origin, node1.websocketUrl, node1.transportUrl, toNode1)
}

func (n *Node) MakeAvailable(worker Worker) {
	go func(worker Worker) {
		m, ok := n.managers.Load(worker.WorkType())
		if ok {
			manager := m.(*WorkManager)
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
	m, ok := n.managers.Load(worker.WorkType())
	if ok {
		manager := m.(*WorkManager)
		if manager.nextAvailableWorker == nil {
			manager.nextAvailableWorker = make(chan Worker)
		}
	} else {
		manager := &WorkManager{
			nextAvailableWorker: make(chan Worker),
			resultChan:          make(chan Work, RESULT_BUFFER_SIZE),
		}
		n.managers.Store(worker.WorkType(), manager)
	}
}

func (n *Node) RemoveWorker(worker Worker) {
	n.Lock()
	defer n.Unlock()
	worker.SetOrgan(nil)
}

func (n *Node) ProcessIncomingWorkRequests(ctx context.Context, connection *Connection) {
	defer connection.Close()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			work := ReceiveWork(connection)
			if n.materialPool != nil && work.workType == diffusion {
				n.ReceiveDiffusion(work)
				continue
			}
			m, ok := n.managers.Load(work.workType)
			if ok {
				manager := m.(*WorkManager)
				if manager.nextAvailableWorker == nil {
					// Ignore: a request we can't handle with the workers we have.
					continue
				}
				if work.status == 0 {
					// Recieved work request.
					ctx, cancel := context.WithTimeout(ctx, WAIT_FOR_WORKER_SEC)
					select {
					case w := <-manager.nextAvailableWorker:
						finishedWork := w.Work(ctx, work)
						SendWork(connection, finishedWork)
						manager.Lock()
						manager.completedCount++
						if finishedWork.status != 200 {
							manager.completedFailureCount++
							// Need more workers to complete the job.
							n.materialPool.PutLigand(&LigandBlob{
								growth: 1,
							})
						}
						manager.Unlock()
					case <-ctx.Done():
						// Didn't have enough workers to process, we need more cells.
						// Signal growth ligand.
						n.materialPool.PutLigand(&LigandBlob{
							growth: 1,
						})
					}
					cancel()
				} else {
					// Received completed work, which is not expected.
					log.Fatalf("Should not receive completed work, got %v", work)
				}
			}
		}
	}
}

func (n *Node) ReceiveDiffusion(request Work) {
	data := &DiffusionSocketData{}
	json.Unmarshal([]byte(request.result), data)

	n.materialPool.PutResource(&ResourceBlob{
		o2:       data.Resources.O2,
		glucose:  data.Resources.Glucose,
		vitamins: data.Resources.Vitamins,
	})
	n.materialPool.PutWaste(&WasteBlob{
		co2:        data.Waste.CO2,
		creatinine: data.Waste.Creatinine,
	})
	n.materialPool.PutHormone(&HormoneBlob{
		granulocyte_csf: data.Hormone.GranulocyteColonyStimulatingFactor,
		macrophage_csf:  data.Hormone.MacrophageColonyStimulatingFactor,
	})
}

func (n *Node) GetNodeStatus(ctx context.Context, connection *Connection) {
	defer connection.Close()
	ticker := time.NewTicker(STATUS_SOCKET_CLOCK_RATE)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var connections []string
			for _, edge := range n.edges {
				address := strings.Replace(edge.workConnection.RemoteAddr().String(), "/work", "", 1)
				connections = append(connections, address)
			}
			var workStatus []WorkStatusData
			n.managers.Range(func(w, m any) bool {
				workType := w.(WorkType)
				manager := m.(*WorkManager)
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
				return true
			})
			materialStatus := MaterialStatusData{
				O2:           n.materialPool.resourcePool.resources.o2,
				Glucose:      n.materialPool.resourcePool.resources.glucose,
				Vitamin:      n.materialPool.resourcePool.resources.vitamins,
				Co2:          n.materialPool.wastePool.wastes.co2,
				Creatinine:   n.materialPool.wastePool.wastes.creatinine,
				Growth:       n.materialPool.ligandPool.ligands.growth,
				Hunger:       n.materialPool.ligandPool.ligands.hunger,
				Asphyxia:     n.materialPool.ligandPool.ligands.asphyxia,
				Inflammation: n.materialPool.ligandPool.ligands.inflammation,
				G_CSF:        n.materialPool.hormonePool.hormones.granulocyte_csf,
				M_CSF:        n.materialPool.hormonePool.hormones.macrophage_csf,
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

func (n *Node) RequestWork(ctx context.Context, request Work) (result Work) {
	ctx, cancel := context.WithTimeout(ctx, TIMEOUT_SEC)
	defer cancel()
	// First try to get a result without sending:
	m, _ := n.managers.LoadOrStore(request.workType, &WorkManager{
		// Skip nextAvailableWorker, to indicate that we can't process
		// requests of this type.
		resultChan: make(chan Work, RESULT_BUFFER_SIZE),
	})
	manager := m.(*WorkManager)
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
		if n.verbose {
			fmt.Printf("%v Received Result: %v\n", n, result)
		}
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
			if n.verbose {
				fmt.Printf("%v Received Result: %v\n", n, result)
			}
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

func (n *Node) SendDiffusion(ctx context.Context) {
	if n.materialPool != nil {
		if len(n.edges) == 0 {
			return
		}
		// Pick a random, valid edge to diffuse to.
		var diffusionEdges []*Edge
		for _, e := range n.edges {
			switch e.edgeType {
			case neuronal:
				// Pass
			default:
				diffusionEdges = append(diffusionEdges, e)
			}
		}
		if len(diffusionEdges) == 0 {
			return
		}
		edge := diffusionEdges[rand.Intn(len(diffusionEdges))]

		// Grab a resource, waste, and hormone blob to diffuse. Can be empty.
		resource := n.materialPool.SplitResource(ctx)
		waste := n.materialPool.SplitWaste(ctx)
		hormone := n.materialPool.SplitHormone(ctx)
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
			Hormone: HormoneBlobData{
				GranulocyteColonyStimulatingFactor: hormone.granulocyte_csf,
				MacrophageColonyStimulatingFactor:  hormone.macrophage_csf,
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

func (n *Node) ProcessIncomingWorkResponses(ctx context.Context, connection *Connection) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			work := ReceiveWork(connection)
			if n.verbose {
				fmt.Printf("%v Received Response: %v\n", n, work)
			}
			m, ok := n.managers.Load(work.workType)
			if ok {
				manager := m.(*WorkManager)
				if work.status == 0 {
					// Received incompleted work, which is not expected.
					log.Fatalf("Should not receive incompleted work, got %v", work)
				} else {
					// Received completed work.
					manager.resultChan <- work
				}
			} else {
				// Received completed work, but not expected.
				log.Fatalf("Received completed work, but unable to process it. Got %v", work)
			}
		}
	}
}
