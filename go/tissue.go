package main

import (
	"bytes"
	"container/ring"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type RenderID string

func MakeRenderId(idPrefix string) RenderID {
	return RenderID(fmt.Sprintf("%v%08v", idPrefix, rand.Intn(100000000)))
}

type Renderable struct {
	id                        RenderID
	visible                   bool
	position                  image.Point
	targetX, targetY, targetZ int
	lastPositions             *ring.Ring
	followId                  RenderID
	ignoreWalls               bool
	renderType                RenderType
}

func (r *Renderable) SetVisible(visible bool) {
	r.visible = visible
}

type InteractionPool struct {
	sync.RWMutex
	interactionMap map[RenderID]CellActor
}

func (p *InteractionPool) Put(r *Renderable, c CellActor) {
	p.Lock()
	defer p.Unlock()
	p.interactionMap[r.id] = c
}

func (p *InteractionPool) Get() CellActor {
	p.RLock()
	defer p.RUnlock()
	for _, c := range p.interactionMap {
		return c
	}
	return nil
}

func (p *InteractionPool) Delete(id RenderID) {
	p.Lock()
	defer p.Unlock()
	delete(p.interactionMap, id)
}

type Tissue struct {
	bounds                image.Rectangle
	cellStreamingChan     chan chan *RenderableSocketData
	cytokineStreamingChan chan chan *RenderableSocketData
	rootMatrix            *ExtracellularMatrix
}

func InitializeTissue(ctx context.Context) *Tissue {
	tissue := &Tissue{
		bounds:                image.Rect(-WORLD_BOUNDS/2, -WORLD_BOUNDS/2, WORLD_BOUNDS/2, WORLD_BOUNDS/2),
		cellStreamingChan:     make(chan chan *RenderableSocketData, STREAMING_BUFFER_SIZE),
		cytokineStreamingChan: make(chan chan *RenderableSocketData, STREAMING_BUFFER_SIZE),
	}
	tissue.BuildTissue()
	return tissue
}

func (t *Tissue) BuildTissue() {
	var curr *ExtracellularMatrix
	for i := 0; i < NUM_PLANES; i++ {
		curr = &ExtracellularMatrix{
			RWMutex: sync.RWMutex{},
			tissue:  t,
			level:   i,
			prev:    curr,
			next:    nil,
			render: &Renderable{
				id:       MakeRenderId("Matrix"),
				visible:  true,
				position: image.Point{0, 0},
				targetX:  0,
				targetY:  0,
				targetZ:  0,
			},
			attached:         make(map[RenderID]*Renderable),
			cytokinesMap:     &sync.Map{},
			interactionsPool: &sync.Map{},
		}
		curr.walls = curr.GenerateWalls(WALL_LINES, WALL_BOXES)
		if curr.prev != nil {
			curr.prev.next = curr
		}
	}
	// Walk back to set next and find root.
	for curr.prev != nil {
		curr = curr.prev
	}
	t.rootMatrix = curr
}

func (t *Tissue) Attach(r *Renderable) {
	t.rootMatrix.Attach(r)
}

func (t *Tissue) Detach(r *Renderable) {
	t.rootMatrix.Detach(r)
}

func (t *Tissue) Start(ctx context.Context) {
	ticker := time.NewTicker(CYTOKINE_TICK_RATE)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.Tick()
		case r := <-t.cellStreamingChan:
			go t.rootMatrix.StartRenderCells(r)
		case r := <-t.cytokineStreamingChan:
			go t.rootMatrix.StartRenderCytokines(r)
		}
	}
}

func (t *Tissue) StreamCells(ctx context.Context, connection *Connection) {
	fmt.Println("Cell render socket opened")
	defer connection.Close()
	go func(c *Connection) {
		defer connection.Close()
		for {
			if _, _, err := c.NextReader(); err != nil {
				fmt.Println("Cell render socket closed")
				return
			}
		}
	}(connection)
	ticker := time.NewTicker(RENDER_STREAM_TICK_RATE)
	for {
		r := make(chan *RenderableSocketData, RENDER_BUFFER_SIZE)
		<-ticker.C
		select {
		case <-ctx.Done():
			return
		case t.cellStreamingChan <- r:
			for renderable := range r {
				out, err := proto.Marshal(renderable)
				if err != nil {
					log.Fatalln("Failed to encode renderable:", err)
				}
				err = connection.WriteMessage(websocket.BinaryMessage, out)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						fmt.Printf("error: %v %v", err, renderable)
					} else {
						fmt.Println("Cell render socket closed")
					}
					return
				}
			}
		}
	}
}

func (t *Tissue) StreamCytokines(ctx context.Context, connection *Connection) {
	fmt.Println("Cytokine render socket opened")
	defer connection.Close()
	go func(c *Connection) {
		defer connection.Close()
		for {
			if _, _, err := c.NextReader(); err != nil {
				fmt.Println("Cytokine render socket closed")
				return
			}
		}
	}(connection)
	ticker := time.NewTicker(RENDER_STREAM_TICK_RATE)
	for {
		r := make(chan *RenderableSocketData, RENDER_BUFFER_SIZE)
		<-ticker.C
		select {
		case <-ctx.Done():
			return
		case t.cytokineStreamingChan <- r:
			for renderable := range r {
				out, err := proto.Marshal(renderable)
				if err != nil {
					log.Fatalln("Failed to encode renderable:", err)
				}
				err = connection.WriteMessage(websocket.BinaryMessage, out)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						fmt.Printf("error: %v %v", err, renderable)
					} else {
						fmt.Println("Cytokine render socket closed")
					}
					return
				}
			}
		}
	}
}

func (t *Tissue) RenderRootMatrix(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	matrix := t.rootMatrix
	buf := new(bytes.Buffer)
	err := png.Encode(buf, matrix)
	if err != nil {
		panic(fmt.Errorf("error while encoding png: %v", err))
	}
	img, err := MakeTitledPng(buf, matrix.RenderMetadata())
	if err != nil {
		panic(fmt.Errorf("error while encoding png: %v", err))
	}
	_, err = w.Write(img.Bytes())
	if err != nil {
		panic(fmt.Errorf("error while sending png: %v", err))
	}
}

func (t *Tissue) Tick() {
	matrix := t.rootMatrix
	for matrix != nil {
		matrix.Tick()
		matrix = matrix.next
	}
}

func (t *Tissue) FindMatrix(r *Renderable) (m *ExtracellularMatrix) {
	found := false
	for m = t.rootMatrix; m != nil && !found; {
		m.RLock()
		_, found = m.attached[r.id]
		m.RUnlock()
		if !found {
			m = m.next
		}
	}
	return m
}

func (t *Tissue) FindRender(renderId RenderID) (r *Renderable) {
	render := &Renderable{id: renderId}
	m := t.FindMatrix(render)
	if m != nil {
		m.RLock()
		r = m.attached[renderId]
		m.RUnlock()
	}
	return
}

func (t *Tissue) Move(r *Renderable) {
	m := t.FindMatrix(r)
	if m == nil {
		return
	}
	var targetRender *Renderable
	if r.followId != "" {
		targetRender = t.FindRender(r.followId)
	} else {
		r.followId = ""
	}
	m.Move(r, targetRender)
}

func (t *Tissue) AddCytokine(r *Renderable, cType CytokineType, concentration uint8) uint8 {
	m := t.FindMatrix(r)
	if m == nil || cType == CytokineType_unknown {
		return 0
	}
	return m.AddCytokine(r.position, cType, concentration)
}

func (t *Tissue) ConsumeCytokines(r *Renderable, cType CytokineType, consumptionRate uint8) uint8 {
	m := t.FindMatrix(r)
	if m == nil {
		return 0
	}
	return m.ConsumeCytokines(r.position, cType, consumptionRate)
}

func (t *Tissue) BroadcastPosition(ctx context.Context, cell CellActor, r *Renderable) {
	m := t.FindMatrix(r)
	if m == nil {
		return
	}
	pool, _ := m.interactionsPool.LoadOrStore(r.position, &InteractionPool{
		interactionMap: map[RenderID]CellActor{},
	})
	pool.(*InteractionPool).Put(r, cell)
}

func (t *Tissue) GetInteractions(ctx context.Context, r *Renderable) (interactions []CellActor) {
	ctx, cancel := context.WithTimeout(ctx, INTERACTIONS_TIMEOUT)
	defer cancel()
	m := t.FindMatrix(r)
	if m == nil {
		return
	}
	x := r.position.X
	x_plus := x + 1
	x_minus := x - 1
	y := r.position.Y
	y_plus := y + 1
	y_minus := y - 1
	points := [9]image.Point{{x_minus, y_plus},
		{x, y_plus},
		{x_plus, y_plus},
		{x_minus, y},
		{x, y},
		{x_plus, y},
		{x_minus, y_minus},
		{x, y_minus},
		{x_plus, y_minus}}
	interactionsChan := make(chan CellActor)
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(interactionsChan)
				return
			default:
				allNil := true
				for i, pt := range points {
					pool, _ := m.interactionsPool.LoadOrStore(pt, &InteractionPool{
						interactionMap: map[RenderID]CellActor{},
					})
					interactionPool := pool.(*InteractionPool)
					if c := interactionPool.Get(); c != nil && c.Organ() != nil && c.Render().position.Eq(points[i]) {
						interactionsChan <- c
					} else if c != nil {
						interactionPool.Delete(c.Render().id)
						allNil = false
					}
				}
				if allNil {
					cancel()
				}
			}
		}
	}()
	for c := range interactionsChan {
		interactions = append(interactions, c)
	}
	return
}

type Walls struct {
	mainStage     Circle
	boundaries    []image.Rectangle
	bubbles       []Circle
	bridges       []Line
	inBoundsCache *sync.Map
}

func (w *Walls) InBounds(pt image.Point) bool {
	i, hasPoint := w.inBoundsCache.Load(pt)
	if hasPoint {
		return i.(bool)
	}
	if w.mainStage.InBounds(pt) {
		w.inBoundsCache.Store(pt, true)
		return true
	}
	for _, b := range w.bridges {
		if b.InBounds(pt) {
			w.inBoundsCache.Store(pt, true)
			return true
		}
	}
	inBounds := false
	for _, b := range w.boundaries {
		if pt.In(b) {
			inBounds = true
		}
	}
	if inBounds {
		for _, b := range w.bubbles {
			if b.InBounds(pt) {
				inBounds = false
			}
		}
	}
	w.inBoundsCache.Store(pt, inBounds)
	return inBounds
}

type ExtracellularMatrix struct {
	sync.RWMutex
	tissue           *Tissue
	level            int
	next             *ExtracellularMatrix
	prev             *ExtracellularMatrix
	walls            *Walls
	attached         map[RenderID]*Renderable
	render           *Renderable
	cytokinesMap     *sync.Map
	interactionsPool *sync.Map
}

func (m *ExtracellularMatrix) ColorModel() color.Model {
	return color.RGBAModel
}

func (m *ExtracellularMatrix) Bounds() image.Rectangle {
	return m.tissue.bounds
}

func (m *ExtracellularMatrix) At(x, y int) color.Color {
	pt := image.Point{x, y}
	if m.walls.InBounds(pt) {
		return color.White
		// return m.GetCytokineColor(pt)
	}
	return color.Black
}

func (m *ExtracellularMatrix) GetCytokineColor(pt image.Point) color.Color {
	cTypes := []CytokineType{
		CytokineType_cell_damage,
		CytokineType_antigen_present,
		CytokineType_induce_chemotaxis,
		CytokineType_cytotoxins,
	}
	concentrations := m.GetCytokineContentrations([]image.Point{pt}, cTypes)[0]
	var hues []float64
	concentration := 0
	for i, t := range cTypes {
		concentration += int(concentrations[i])
		if concentration > 0 {
			var h float64
			switch t {
			case CytokineType_cell_damage:
				// Red.
				h = 0
			case CytokineType_cytotoxins:
				// Pink.
				h = float64(280) / float64(360)
			case CytokineType_antigen_present:
				// Orange.
				h = float64(25) / float64(360)
			case CytokineType_induce_chemotaxis:
				// Green.
				h = float64(125) / float64(360)
			}
			hues = append(hues, h)
		}
	}
	if len(hues) > 0 {
		l := 1 - 0.5*float64(concentration)/float64(math.MaxInt8)
		h := hues[0]
		r, g, b := HSLtoRGB(h, 1, l)
		return color.RGBA{r, g, b, math.MaxUint8}
	} else {
		return color.White
	}
}

func (m *ExtracellularMatrix) RenderMetadata() string {
	return fmt.Sprintf("{\"z\":\"%02v\",\"id\":\"%v\"}", m.level, string(m.render.id))
}

func (m *ExtracellularMatrix) ConstrainBounds(r *Renderable) {
	b := m.tissue.bounds
	if r.position.X > b.Max.X {
		r.position.X = b.Max.X
	}
	if r.position.X < b.Min.X {
		r.position.X = b.Min.X
	}
	if r.position.Y > b.Max.Y {
		r.position.Y = b.Max.Y
	}
	if r.position.Y < b.Min.Y {
		r.position.Y = b.Min.Y
	}
}

func (m *ExtracellularMatrix) ConstrainTargetBounds(r *Renderable) {
	b := m.tissue.bounds
	if r.targetX > b.Max.X {
		r.targetX = b.Max.X
	}
	if r.targetX < b.Min.X {
		r.targetX = b.Min.X
	}
	if r.targetY > b.Max.Y {
		r.targetY = b.Max.Y
	}
	if r.targetY < b.Min.Y {
		r.targetY = b.Min.Y
	}
	if r.targetZ < 0 {
		r.targetZ = 0
	}
	if r.targetZ > NUM_PLANES-1 {
		r.targetZ = NUM_PLANES - 1
	}
}

func (m *ExtracellularMatrix) Physics(r *Renderable) {
	if !m.walls.InBounds(r.position) && !r.ignoreWalls {
		dx := 0
		dy := 0
		if r.position.X > m.walls.mainStage.center.X {
			dx = -1
		}
		if r.position.X < m.walls.mainStage.center.X {
			dx = 1
		}
		if r.position.Y > m.walls.mainStage.center.Y {
			dy = -1
		}
		if r.position.Y < m.walls.mainStage.center.Y {
			dy = 1
		}
		m.MoveX(r, dx)
		m.MoveY(r, dy)
	}
}

func (m *ExtracellularMatrix) Move(r *Renderable, targetRender *Renderable) {
	m.ConstrainTargetBounds(r)

	targetX := r.targetX
	targetY := r.targetY
	targetZ := r.targetZ

	if targetRender != nil {
		m.ConstrainTargetBounds(targetRender)
		targetX = targetRender.position.X
		targetY = targetRender.position.Y
		targetZ = targetRender.targetZ
	}

	if targetX > r.position.X {
		m.MoveX(r, 1)
	}
	if targetX < r.position.X {
		m.MoveX(r, -1)
	}
	if targetY > r.position.Y {
		m.MoveY(r, 1)
	}
	if targetY < r.position.Y {
		m.MoveY(r, -1)
	}
	if targetZ > m.level {
		m.MoveUp(r)
	}
	if targetZ < m.level {
		m.MoveDown(r)
	}
	m.Physics(r)
}

func (m *ExtracellularMatrix) MoveX(r *Renderable, x int) {
	r.position.X += x
	m.ConstrainBounds(r)
	if r.lastPositions.Len() > 0 {
		r.lastPositions = r.lastPositions.Next()
	}
	r.lastPositions.Value = r.position
}

func (m *ExtracellularMatrix) MoveY(r *Renderable, y int) {
	r.position.Y += y
	m.ConstrainBounds(r)
	r.lastPositions.Value = r.position
	r.lastPositions = r.lastPositions.Next()
}

func (m *ExtracellularMatrix) MoveUp(r *Renderable) {
	if m.prev != nil {
		m.Detach(r)
		m.prev.Attach(r)
		m.prev.ConstrainBounds(r)
	}
}

func (m *ExtracellularMatrix) MoveDown(r *Renderable) {
	if m.next != nil {
		m.Detach(r)
		m.next.Attach(r)
		m.next.ConstrainBounds(r)
	}
}

func (m *ExtracellularMatrix) Attach(r *Renderable) {
	m.Lock()
	defer m.Unlock()
	m.attached[r.id] = r
}

func (m *ExtracellularMatrix) Detach(r *Renderable) {
	m.Lock()
	defer m.Unlock()
	delete(m.attached, r.id)
}

func (m *ExtracellularMatrix) Tick() {
	m.cytokinesMap.Range(func(_, cytokines any) bool {
		cytokines.(*sync.Map).Range(func(_, c any) bool {
			c.(*Cytokine).Tick()
			return true
		})
		return true
	})
	m.RLock()
	var renderables []*Renderable
	for _, r := range m.attached {
		renderables = append(renderables, r)
	}
	m.RUnlock()
	for _, r := range renderables {
		m.Physics(r)
	}
}

func (m *ExtracellularMatrix) StartRenderCells(renderChan chan *RenderableSocketData) {
	m.RLock()
	var attached []*Renderable
	for _, a := range m.attached {
		attached = append(attached, a)
	}
	m.RUnlock()
	for _, a := range attached {
		renderChan <- m.RenderCells(a)
	}
	close(renderChan)
}

func (m *ExtracellularMatrix) StartRenderCytokines(renderChan chan *RenderableSocketData) {
	for _, r := range m.RenderCytokines() {
		renderChan <- r
	}
	close(renderChan)
}

func (m *ExtracellularMatrix) RenderCells(r *Renderable) *RenderableSocketData {
	return &RenderableSocketData{
		Id:      string(r.id),
		Visible: r.visible,
		Position: &Position{
			X: int32(r.position.X),
			Y: int32(r.position.Y),
			Z: int32(m.level),
		},
		Type: &r.renderType,
	}
}

func (m *ExtracellularMatrix) RenderCytokines() (socketData []*RenderableSocketData) {
	m.cytokinesMap.Range(func(pt any, cytokines any) bool {
		var point = pt.(image.Point)
		cytokines.(*sync.Map).Range(func(pt any, c any) bool {
			var cytokine = c.(*Cytokine)
			if cytokine.concentration > 0 {
				var cytokineRender = &RenderableSocketData{
					Id:      fmt.Sprintf("Cytokine%v-%v-%v", cytokine.cytokine, point.X, point.Y),
					Visible: true,
					Position: &Position{
						X: int32(point.X),
						Y: int32(point.Y),
						Z: int32(m.level),
					},
					Type: &RenderType{
						Type: &RenderType_CytokineType{
							CytokineType: cytokine.cytokine,
						},
					},
				}
				socketData = append(socketData, cytokineRender)
			}
			return true
		})
		return true
	})
	return
}

func (m *ExtracellularMatrix) GetOpenSpaces(pts []image.Point) (open []image.Point) {
	for _, pt := range pts {
		if m.walls.InBounds(pt) && pt.In(m.Bounds()) {
			open = append(open, pt)
		}
	}
	return
}

func (m *ExtracellularMatrix) GetCytokineContentrations(pts []image.Point, types []CytokineType) (concentrations [][]uint8) {
	for i := len(pts); i > 0; i-- {
		concentrations = append(concentrations, make([]uint8, len(types)))
	}
	m.cytokinesMap.Range(func(_, cytokines any) bool {
		for j, t := range types {
			c, hasCytokine := cytokines.(*sync.Map).Load(t)
			if hasCytokine {
				cytokine := c.(*Cytokine)
				for i, pt := range pts {
					if m.walls.InBounds(pt) {
						cn := concentrations[i]
						concentration := int(cn[j]) + int(cytokine.At(pt))
						if concentration > math.MaxUint8 {
							cn[j] = math.MaxUint8
						} else {
							cn[j] = uint8(concentration)
						}
					}
				}
			}
		}
		return true
	})
	return
}

func (m *ExtracellularMatrix) GetCytokinesAtPoint(pt image.Point) *sync.Map {
	cytokines, _ := m.cytokinesMap.LoadOrStore(pt, &sync.Map{})
	return cytokines.(*sync.Map)
}

func (m *ExtracellularMatrix) AddCytokine(pt image.Point, t CytokineType, concentration uint8) uint8 {
	if t == CytokineType_unknown {
		return 0
	}
	cytokines := m.GetCytokinesAtPoint(pt)
	c, _ := cytokines.LoadOrStore(t, MakeCytokine(pt, t, concentration))
	return c.(*Cytokine).Add(concentration)
}

func (m *ExtracellularMatrix) GetCytokinesWithinRange(pt image.Point, t CytokineType) (cytokinesInRange []*Cytokine) {
	m.cytokinesMap.Range(func(_, cytokines any) bool {
		c, hasCytokine := cytokines.(*sync.Map).Load(t)
		if hasCytokine {
			cytokine := c.(*Cytokine)
			if cytokine.InBounds(pt) {
				cytokinesInRange = append(cytokinesInRange, cytokine)
			}
		}
		return true
	})
	return
}

func (m *ExtracellularMatrix) ConsumeCytokines(pt image.Point, t CytokineType, consumptionRate uint8) uint8 {
	cytokines := m.GetCytokinesWithinRange(pt, t)
	consumed := uint8(0)
	for _, c := range cytokines {
		consumed += c.Sub(consumptionRate)
		if consumed == math.MaxUint8 {
			return consumed
		}
	}
	return consumed
}

func (m *ExtracellularMatrix) GenerateWalls(numLines int, numBoxesPerLine int) *Walls {
	bounds := m.tissue.bounds
	var boundaries []image.Rectangle
	for i := 0; i < numLines; i++ {
		p0 := MakeRandPoint(bounds)
		for j := 0; j < numBoxesPerLine; j++ {
			p1 := MakeRandPoint(bounds)
			l := Line{p0, p1, LINE_WIDTH}
			p2 := l.GetRandPoint()
			boundary := MakeRandRect(bounds)
			if !p2.In(boundary) {
				boundary = boundary.Add(boundary.Min.Sub(p2)).Intersect(bounds)
			}
			boundaries = append(boundaries, boundary)
			p0 = p1
		}
	}
	var finalBoundaries []image.Rectangle
	var bubbles []Circle
	var bridges []Line
	for _, boundary := range boundaries {
		hasOverlap := false
		for _, checkboundary := range boundaries {
			if boundary != checkboundary && checkboundary.Overlaps(boundary) {
				hasOverlap = true
			}
		}
		if hasOverlap {
			finalBoundaries = append(finalBoundaries, boundary)
			minX := boundary.Dx() / 2
			if minX > MAX_RADIUS {
				minX = MAX_RADIUS
			}
			minY := boundary.Dy() / 2
			if minY > MAX_RADIUS {
				minY = MAX_RADIUS
			}
			bubbles = append(bubbles, Circle{boundary.Min, RandInRange(2, minX)})
			bubbles = append(bubbles, Circle{boundary.Max, RandInRange(2, minY)})
		}
	}
	mainStage := Circle{image.Point{RandInRange(-5, 5), RandInRange(-5, 5)}, MAIN_STAGE_RADIUS}
	return &Walls{
		mainStage:     mainStage,
		boundaries:    finalBoundaries,
		bubbles:       bubbles,
		bridges:       bridges,
		inBoundsCache: &sync.Map{},
	}
}
