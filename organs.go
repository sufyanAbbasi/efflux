package main

import (
	"bytes"
	"container/ring"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type Worker interface {
	WorkType() WorkType
	SetOrgan(organ *Node)
	Work(ctx context.Context, request Work) Work
	IsApoptosis() bool
}

type Work struct {
	workType WorkType
	result   string
	status   int
}

type TransportRequest struct {
	Name            string
	Base            []byte
	DNAType         int // Curve number, like elliptic.P384()
	CellType        CellType
	WorkType        WorkType
	ParentRenderID  string
	SpawnTime       time.Time
	TransportPath   [10]string
	WantPath        [10]string
	MHC_II_Proteins []Protein
}

type EdgeType int

const (
	cardiovascular EdgeType = iota
	neuronal
	lymphatic
	muscular
	skeletal
	gut_lining
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
	spawnTime time.Time,
	transportPath [10]string,
	wantPath [10]string,
	mhc_ii map[Protein]bool,
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
	var mhc_ii_proteins []Protein
	for protein := range mhc_ii {
		mhc_ii_proteins = append(mhc_ii_proteins, protein)
	}
	jsonData, err := json.Marshal(TransportRequest{
		Name:            name,
		Base:            dnaBase,
		DNAType:         dnaType,
		CellType:        cellType,
		WorkType:        workType,
		ParentRenderID:  parentRenderID,
		SpawnTime:       spawnTime,
		TransportPath:   transportPath,
		WantPath:        wantPath,
		MHC_II_Proteins: mhc_ii_proteins,
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
		log.Fatal(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if cell.CellType() == CellType_ViralLoadCarrier {
		virusCarrier := cell.(*VirusCarrier)
		n.antigenPool.DepositViralLoad(&ViralLoad{
			virus:         virusCarrier.virus,
			concentration: VIRAL_LOAD_CARRIER_CONCENTRATION,
		})
		if n.verbose {
			fmt.Println("Viral load:", cell, "added to", n)
		}
	} else {
		cell.SetOrgan(n)
		cell.RecordTransport()
		if n.verbose {
			fmt.Println("Spawned:", cell, "in", cell.Organ())
		}
		ctx, stop := context.WithCancel(ctx)
		cell.SetStop(stop)
		cell.Start(ctx)
	}

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
		// Place cell somewhere random.
		render = &Renderable{
			position: image.Point{RandInRange(-MAIN_STAGE_RADIUS, MAIN_STAGE_RADIUS), RandInRange(-MAIN_STAGE_RADIUS, MAIN_STAGE_RADIUS)},
		}
	}
	return MakeCellFromType(request.CellType, request.WorkType, dna, render, request.SpawnTime, request.TransportPath, request.WantPath, request.MHC_II_Proteins), nil
}

func SendWork(connection *Connection, request Work, diffusion *DiffusionSocketData) {
	work := &WorkSocketData{
		WorkType:  int32(request.workType),
		Result:    request.result,
		Status:    int32(request.status),
		Diffusion: diffusion,
	}
	out, err := proto.Marshal(work)
	if err != nil {
		log.Fatalln("Failed to encode work:", err)
	}
	err = connection.WriteMessage(websocket.BinaryMessage, out)
	if err != nil {
		log.Fatal("Send: ", err)
	}
}

func SendStatus(connection *Connection, status *StatusSocketData) error {
	out, err := proto.Marshal(status)
	if err != nil {
		return err
	}
	err = connection.WriteMessage(websocket.BinaryMessage, out)
	if err != nil {
		return err
	}
	return nil
}

func ReceiveWork(connection *Connection) (request Work, diffusion *DiffusionSocketData) {
	_, message, err := connection.ReadMessage()
	if err != nil {
		log.Fatalln("Receive: ", err)
	}
	work := &WorkSocketData{}
	err = proto.Unmarshal(message, work)
	if err != nil {
		log.Fatalln("Failed to parse work: ", err)
	}
	request.workType = WorkType(work.WorkType)
	request.status = int(work.Status)
	request.result = work.Result
	diffusion = work.Diffusion
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
	name           string
	edges          []*Edge
	serverMux      *http.ServeMux
	origin         string
	port           string
	websocketUrl   string
	transportUrl   string
	managers       *sync.Map
	nanobotManager *NanobotManager
	materialPool   *MaterialPool
	antigenPool    *AntigenPool
	tissue         *Tissue
	verbose        bool
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
	node.antigenPool = InitializeAntigenPool(ctx)
	node.nanobotManager = InitializeNanobotManager(ctx)
	graph.allNodes[url] = node
	node.Start(ctx)
	go node.tissue.Start(ctx)
	go node.nanobotManager.Start(ctx)
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

func (c *Connection) WriteMessage(messageType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.Conn.WriteMessage(messageType, data)
}

func (c *Connection) ReadMessage() (messageType int, p []byte, err error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	return c.Conn.ReadMessage()
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
	n.serverMux.HandleFunc(INTERACTIONS_LOGIN_ENDPOINT, func(w http.ResponseWriter, r *http.Request) {
		n.InteractionsLogin(ctx, w, r)
	})
	n.serverMux.HandleFunc(INTERACTIONS_STREAM_ENDPOINT, WebsocketHandler(ctx, n.InteractionStream))
	if n.tissue != nil {
		n.serverMux.HandleFunc(WORLD_RENDER_ENDPOINT, WebsocketHandler(ctx, n.tissue.Stream))
		n.serverMux.HandleFunc(WORLD_TEXTURE_ENDPOINT, n.tissue.RenderRootMatrix)
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

func (n *Node) InteractionsLogin(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	loginRequest := &InteractionLoginRequest{}
	message, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request was malformed."))
		return
	}
	err = proto.Unmarshal(message, loginRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request was malformed."))
		return
	}
	token, err := uuid.Parse(loginRequest.GetSessionToken())
	if err != nil {
		// Request UUID was non-existent malformed, create a new one.
		token, err = uuid.NewRandom()
		if err != nil {
			panic("Unable to generate session token")
		}
	}
	name := fmt.Sprint(time.Now().Unix())
	nanobot_, _ := n.nanobotManager.nanobots.LoadOrStore(token, &Nanobot{
		name:         name,
		sessionToken: token,
		render: &Renderable{
			id:            MakeRenderId("Nanobot"),
			visible:       false,
			position:      image.Point{},
			targetX:       0,
			targetY:       0,
			targetZ:       0,
			lastPositions: &ring.Ring{},
		},
	})
	nanobot := nanobot_.(*Nanobot)
	nanobot.RenewExpiry()
	nanobot.organ = n
	nanobot.Start(ctx)
	data, err := proto.Marshal(&InteractionLoginResponse{
		SessionToken: token.String(),
		Expiry:       int32(nanobot.expiry.Unix()),
		RenderId:     string(nanobot.render.id),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to login right now"))
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

func (n *Node) InteractionStream(ctx context.Context, connection *Connection) {
	fmt.Println("Interaction socket opened")
	defer connection.Close()
	for {
		_, r, err := connection.NextReader()
		if err != nil {
			fmt.Println("Interaction socket closed", err)
			return
		}
		data, err := io.ReadAll(r)
		response := &InteractionResponse{
			Status:       InteractionResponse_success,
			ErrorMessage: "",
		}
		if err != nil {
			message := "Unable to parse interaction request"
			fmt.Println(message, err)
			response.Status = InteractionResponse_failure
			response.ErrorMessage = message
		}
		request := &InteractionRequest{}
		err = proto.Unmarshal(data, request)
		if err != nil {
			message := "Unable to parse interaction request"
			fmt.Println(message, err)
			response.Status = InteractionResponse_failure
			response.ErrorMessage = message
		}
		token, err := uuid.Parse(request.SessionToken)
		nanobot, ok := n.nanobotManager.nanobots.Load(token)
		if ok {
			toClose, err := nanobot.(*Nanobot).ProcessInteraction(request)
			if err != nil {
				fmt.Println("", err)
				response.Status = InteractionResponse_failure
				response.ErrorMessage = fmt.Sprint(err)
			}
			if toClose {
				b := nanobot.(*Nanobot)
				n.nanobotManager.nanobots.Delete(b.sessionToken)
				b.CleanUp()
			}
		} else {
			message := "Invalid session token, please refresh"
			fmt.Println(message, err)
			response.Status = InteractionResponse_failure
			response.ErrorMessage = message
		}
		out, err := proto.Marshal(response)
		if err != nil {
			fmt.Println("Failed to encode renderable:", err)
			response.Status = InteractionResponse_failure
			response.ErrorMessage = "Internal error occured"
		}
		err = connection.WriteMessage(websocket.BinaryMessage, out)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("error: %v %v", err, response)
			} else {
				fmt.Println("Interaction socket closed")
			}
			return
		}

	}
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

func (n *Node) MakeAvailable(ctx context.Context, worker Worker) {
	go func(worker Worker) {
		m, ok := n.managers.Load(worker.WorkType())
		if ok && !worker.IsApoptosis() {
			manager := m.(*WorkManager)
			if manager.nextAvailableWorker == nil {
				manager.nextAvailableWorker = make(chan Worker)
			}
			select {
			case <-ctx.Done():
				return
			case manager.nextAvailableWorker <- worker:
			}
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
			log.Fatalln("Node cannot process incoming work")
			return
		default:
			work, diffusion := ReceiveWork(connection)
			if n.materialPool != nil && work.workType == WorkType_diffusion {
				n.ReceiveDiffusion(diffusion)
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
						SendWork(connection, finishedWork, nil)
						manager.Lock()
						manager.completedCount++
						if finishedWork.status != 200 {
							manager.completedFailureCount++
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

func (n *Node) ReceiveDiffusion(data *DiffusionSocketData) {
	n.materialPool.PutResource(&ResourceBlob{
		o2:       int(data.Resources.O2),
		glucose:  int(data.Resources.Glucose),
		vitamins: int(data.Resources.Vitamins),
	})
	n.materialPool.PutWaste(&WasteBlob{
		co2:        int(data.Waste.CO2),
		creatinine: int(data.Waste.Creatinine),
	})
	n.materialPool.PutHormone(&HormoneBlob{
		granulocyte_csf: int(data.Hormone.GranulocyteColonyStimulatingFactor),
		macrophage_csf:  int(data.Hormone.MacrophageColonyStimulatingFactor),
		interleukin_3:   int(data.Hormone.Interleukin3),
		interleukin_2:   int(data.Hormone.Interleukin2),
	})
	n.antigenPool.PutDiffusionLoad(data.Antigen)
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
			var workStatus []*WorkStatusSocketData
			n.managers.Range(func(w, m any) bool {
				workType := w.(WorkType)
				manager := m.(*WorkManager)
				manager.Lock()
				workStatus = append(workStatus, &WorkStatusSocketData{
					WorkType:              workType.String(),
					RequestCount:          int32(manager.requestCount),
					SuccessCount:          int32(manager.successCount),
					FailureCount:          int32(manager.failureCount),
					CompletedCount:        int32(manager.completedCount),
					CompletedFailureCount: int32(manager.completedFailureCount),
				})
				manager.requestCount = 0
				manager.successCount = 0
				manager.failureCount = 0
				manager.completedCount = 0
				manager.completedFailureCount = 0
				manager.Unlock()
				return true
			})
			materialStatus := &MaterialStatusSocketData{
				O2:           int32(n.materialPool.resourcePool.resources.o2),
				Glucose:      int32(n.materialPool.resourcePool.resources.glucose),
				Vitamin:      int32(n.materialPool.resourcePool.resources.vitamins),
				Co2:          int32(n.materialPool.wastePool.wastes.co2),
				Creatinine:   int32(n.materialPool.wastePool.wastes.creatinine),
				Growth:       int32(n.materialPool.ligandPool.ligands.growth),
				Hunger:       int32(n.materialPool.ligandPool.ligands.hunger),
				Asphyxia:     int32(n.materialPool.ligandPool.ligands.asphyxia),
				Inflammation: int32(n.materialPool.ligandPool.ligands.inflammation),
				GCsf:         int32(n.materialPool.hormonePool.hormones.granulocyte_csf),
				MCsf:         int32(n.materialPool.hormonePool.hormones.macrophage_csf),
				Il_3:         int32(n.materialPool.hormonePool.hormones.interleukin_3),
				Il_2:         int32(n.materialPool.hormonePool.hormones.interleukin_2),
				ViralLoad:    int32(n.antigenPool.GetViralLoad()),
				AntibodyLoad: int32(n.antigenPool.GetAntibodyLoad()),
			}
			err := SendStatus(connection, &StatusSocketData{
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
			SendWork(edge.workConnection, request, nil)
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
		diffusionData := &DiffusionSocketData{
			Resources: &ResourceBlobSocketData{
				O2:       int32(resource.o2),
				Glucose:  int32(resource.glucose),
				Vitamins: int32(resource.vitamins),
			},
			Waste: &WasteBlobSocketData{
				CO2:        int32(waste.co2),
				Creatinine: int32(waste.creatinine),
			},
			Hormone: &HormoneBlobSocketData{
				GranulocyteColonyStimulatingFactor: int32(hormone.granulocyte_csf),
				MacrophageColonyStimulatingFactor:  int32(hormone.macrophage_csf),
				Interleukin3:                       int32(hormone.interleukin_3),
				Interleukin2:                       int32(hormone.interleukin_2),
			},
			Antigen: n.antigenPool.GetDiffusionLoad(),
		}
		SendWork(edge.workConnection, Work{
			workType: WorkType_diffusion,
			status:   0,
		}, diffusionData)

		// If there are viral loads that are greater than max, deposit it.
		for _, viralLoad := range n.antigenPool.GetExcessViralLoad() {
			virusDNA := viralLoad.virus.dna
			fmt.Println("Viral load diffused to", edge.transportUrl)
			MakeTransportRequest(edge.transportUrl, virusDNA.name, virusDNA, CellType_ViralLoadCarrier, WorkType_nothing, "", time.Now(), [10]string{}, [10]string{}, nil)
		}
	}
}

func (n *Node) ProcessIncomingWorkResponses(ctx context.Context, connection *Connection) {
	for {
		select {
		case <-ctx.Done():
			log.Fatalln("Node cannot process work responses")
			return
		default:
			work, _ := ReceiveWork(connection)
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
