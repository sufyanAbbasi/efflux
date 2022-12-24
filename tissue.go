package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
}

type RenderableData struct {
	Id      RenderID `json:"id"`
	Visible bool     `json:"visible"`
	X       int      `json:"x"`
	Y       int      `json:"y"`
	Z       int      `json:"z"`
}

func (r *Renderable) SetVisible(visible bool) {
	r.visible = visible
}

type Tissue struct {
	ctx           context.Context
	bounds        image.Rectangle
	streamingChan chan chan RenderableData
	rootMatrix    *ExtracellularMatrix
}

func InitializeTissue(ctx context.Context) *Tissue {
	tissue := &Tissue{
		ctx:           ctx,
		bounds:        image.Rect(-WORLD_BOUNDS/2, -WORLD_BOUNDS/2, WORLD_BOUNDS/2, WORLD_BOUNDS/2),
		streamingChan: make(chan chan RenderableData),
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
			attached:     make(map[RenderID]*Renderable),
			cytokinesMap: make(map[image.Point]Cytokines),
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
		case r := <-t.streamingChan:
			go t.rootMatrix.Render(r)
		}
	}
}

func (t *Tissue) Stream(connection *Connection) {
	fmt.Println("Render socket opened")
	defer connection.Close()
	go func(c *Connection) {
		defer connection.Close()
		for {
			if _, _, err := c.NextReader(); err != nil {
				fmt.Println("Render socket closed")
				return
			}
		}
	}(connection)
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			t.StreamMatrices(connection)
			r := make(chan RenderableData)
			t.streamingChan <- r
			for renderable := range r {
				err := connection.WriteJSON(renderable)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						fmt.Printf("error: %v", err)
					} else {
						fmt.Println("Render socket closed")
					}
					return
				}
			}
		}
	}
}

func (t *Tissue) StreamMatrices(connection *Connection) {
	matrix := t.rootMatrix
	for matrix != nil {
		buf := new(bytes.Buffer)
		err := png.Encode(buf, matrix)
		if err != nil {
			fmt.Printf("Error while encoding png: %v", err)
			return
		}
		img, err := MakeTitledPng(buf, matrix.RenderMetadata())
		if err != nil {
			fmt.Printf("Error while encoding png: %v", err)
			return
		}
		err = connection.WriteMessage(websocket.BinaryMessage, img.Bytes())
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("error: %v", err)
			} else {
				fmt.Println("Render socket closed")
			}
			return
		}
		matrix = matrix.next
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

func (t *Tissue) Move(r *Renderable) {
	m := t.FindMatrix(r)
	if m == nil {
		return
	}
	m.Move(r)
}

func (t *Tissue) AddCytokine(r *Renderable, cType CytokineType, concentration uint8) uint8 {
	m := t.FindMatrix(r)
	if m == nil {
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

type Walls struct {
	sync.RWMutex
	mainStage     Circle
	boundaries    []image.Rectangle
	bubbles       []Circle
	bridges       []Line
	inBoundsCache map[image.Point]bool
}

func (w *Walls) InBounds(pt image.Point) bool {
	w.RLock()
	inBounds, hasPoint := w.inBoundsCache[pt]
	w.RUnlock()
	if hasPoint {
		return inBounds
	}
	if w.mainStage.InBounds(pt) {
		w.RLock()
		w.inBoundsCache[pt] = true
		w.RUnlock()
		return true
	}
	for _, b := range w.bridges {
		if b.InBounds(pt) {
			w.RLock()
			w.inBoundsCache[pt] = true
			w.RUnlock()
			return true
		}
	}
	inBounds = false
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
	w.RLock()
	w.inBoundsCache[pt] = inBounds
	w.RUnlock()
	return inBounds
}

type ExtracellularMatrix struct {
	sync.RWMutex
	tissue       *Tissue
	level        int
	next         *ExtracellularMatrix
	prev         *ExtracellularMatrix
	walls        *Walls
	cytokinesMap map[image.Point]Cytokines
	cytokineMu   sync.RWMutex
	render       *Renderable
	attached     map[RenderID]*Renderable
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
		pts := []image.Point{pt}
		cellDamageConcentration := m.GetCytokineContentrations(pts, cell_damage)[0]
		chemotaxisConcentration := m.GetCytokineContentrations(pts, induce_chemotaxis)[0]

		if chemotaxisConcentration > 0 && cellDamageConcentration > 0 {
			// Randomly pick one or the other to show.
			if rand.Intn(2) == 0 {
				return color.RGBA{math.MaxUint8, math.MaxUint8 - uint8(cellDamageConcentration), math.MaxUint8 - uint8(cellDamageConcentration), math.MaxUint8}
			} else {
				return color.RGBA{math.MaxUint8 - uint8(chemotaxisConcentration), math.MaxUint8, math.MaxUint8 - uint8(chemotaxisConcentration), math.MaxUint8}
			}
		}
		if cellDamageConcentration > 0 {
			return color.RGBA{math.MaxUint8, math.MaxUint8 - uint8(cellDamageConcentration), math.MaxUint8 - uint8(cellDamageConcentration), math.MaxUint8}
		}
		if chemotaxisConcentration > 0 {
			return color.RGBA{math.MaxUint8 - uint8(chemotaxisConcentration), math.MaxUint8, math.MaxUint8 - uint8(chemotaxisConcentration), math.MaxUint8}
		}
		return color.White
	}

	return color.Black
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
	if !m.walls.InBounds(r.position) {
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

func (m *ExtracellularMatrix) Move(r *Renderable) {
	m.ConstrainTargetBounds(r)
	if r.targetX > r.position.X {
		m.MoveX(r, 1)
	}
	if r.targetX < r.position.X {
		m.MoveX(r, -1)
	}
	if r.targetY > r.position.Y {
		m.MoveY(r, 1)
	}
	if r.targetY < r.position.Y {
		m.MoveY(r, -1)
	}
	if r.targetZ > m.level {
		m.MoveUp(r)
	}
	if r.targetZ < m.level {
		m.MoveDown(r)
	}
	m.Physics(r)
}

func (m *ExtracellularMatrix) MoveX(r *Renderable, x int) {
	r.position.X += x
	m.ConstrainBounds(r)
}

func (m *ExtracellularMatrix) MoveY(r *Renderable, y int) {
	r.position.Y += y
	m.ConstrainBounds(r)
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
	m.cytokineMu.RLock()
	for _, cytokines := range m.cytokinesMap {
		for _, c := range cytokines {
			c.Tick()
		}
	}
	m.cytokineMu.RUnlock()
	m.RLock()
	var renderables []*Renderable
	for _, r := range m.attached {
		renderables = append(renderables, r)
	}
	for _, r := range renderables {
		m.Physics(r)
	}
	m.RUnlock()
}

func (m *ExtracellularMatrix) Render(renderChan chan RenderableData) {
	m.RLock()
	var attached []*Renderable
	for _, a := range m.attached {
		attached = append(attached, a)
	}
	m.RUnlock()
	for _, a := range attached {
		renderChan <- m.RenderObject(a)
	}
	close(renderChan)
}

func (m *ExtracellularMatrix) RenderObject(r *Renderable) RenderableData {
	return RenderableData{
		Id:      r.id,
		Visible: r.visible,
		X:       r.position.X,
		Y:       r.position.Y,
		Z:       m.level,
	}
}

func (m *ExtracellularMatrix) GetCytokineContentrations(pts []image.Point, t CytokineType) (concentrations []uint8) {
	m.cytokineMu.RLock()
	defer m.cytokineMu.RUnlock()
	for i := len(pts); i > 0; i-- {
		concentrations = append(concentrations, 0)
	}
	for _, cytokines := range m.cytokinesMap {
		c, hasCytokine := cytokines[t]
		if hasCytokine {
			for i, pt := range pts {
				if m.walls.InBounds(pt) {
					concentration := int(concentrations[i]) + int(c.At(pt))
					if concentration > math.MaxUint8 {
						concentrations[i] = math.MaxUint8
					} else {
						concentrations[i] = uint8(concentration)
					}
				}
			}
		}
	}
	return
}

func (m *ExtracellularMatrix) GetCytokinesAtPoint(pt image.Point) Cytokines {
	m.cytokineMu.Lock()
	defer m.cytokineMu.Unlock()
	cytokines, hasPoint := m.cytokinesMap[pt]
	if !hasPoint {
		cytokines = make(map[CytokineType]*Cytokine)
		m.cytokinesMap[pt] = cytokines
	}
	return cytokines
}

func (m *ExtracellularMatrix) AddCytokine(pt image.Point, t CytokineType, concentration uint8) uint8 {
	cytokines := m.GetCytokinesAtPoint(pt)
	m.cytokineMu.Lock()
	c, hasCytokine := cytokines[t]
	if !hasCytokine {
		c = MakeCytokine(pt, t, concentration)
		cytokines[t] = c
	}
	m.cytokineMu.Unlock()
	return c.Add(concentration)
}

func (m *ExtracellularMatrix) GetCytokinesWithinRange(pt image.Point, t CytokineType) (cytokinesInRange []*Cytokine) {
	m.cytokineMu.RLock()
	defer m.cytokineMu.RUnlock()
	for _, cytokines := range m.cytokinesMap {
		c, hasCytokine := cytokines[t]
		if hasCytokine && c.InBounds(pt) {
			cytokinesInRange = append(cytokinesInRange, c)
		}
	}
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
		inBoundsCache: make(map[image.Point]bool),
	}
}
