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
	id      RenderID
	visible bool
	x, y    int
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
				id:      MakeRenderId("Matrix"),
				visible: true,
				x:       0,
				y:       0,
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

func (t *Tissue) MakeNewRenderAndAttach(idPrefix string) *Renderable {
	render := &Renderable{
		id:      MakeRenderId(idPrefix),
		visible: true,
		x:       0,
		y:       0,
	}
	t.Attach(render)
	return render
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
		matrix.cytokineMu.RLock()
		for _, cytokines := range matrix.cytokinesMap {
			for _, c := range cytokines {
				c.Tick()
			}
		}
		matrix.cytokineMu.RUnlock()
		matrix = matrix.next
	}
}

type Walls struct {
	boundaries []image.Rectangle
	bubbles    []Circle
	bridges    []Line
}

func (w *Walls) InBounds(x, y int) bool {
	pt := image.Pt(x, y)
	for _, b := range w.bridges {
		if b.InBounds(pt) {
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
	if m.walls.InBounds(x, y) {
		concentration := 0
		m.cytokineMu.RLock()
		for _, cytokines := range m.cytokinesMap {
			c, hasCytokine := cytokines[cell_damage]
			if hasCytokine {
				concentration += int(c.At(image.Point{x, y}))

			}
		}
		m.cytokineMu.RUnlock()
		if concentration > 0 {
			if concentration > math.MaxUint8 {
				concentration = math.MaxUint8
			}
			return color.RGBA{math.MaxUint8, math.MaxUint8 - uint8(concentration), math.MaxUint8 - uint8(concentration), math.MaxUint8}
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
	if r.x > b.Max.X {
		r.x = b.Max.X
	}
	if r.x < b.Min.X {
		r.x = b.Min.X
	}
	if r.y > b.Max.Y {
		r.y = b.Max.Y
	}
	if r.y < b.Min.Y {
		r.y = b.Min.Y
	}
}

func (m *ExtracellularMatrix) MoveX(r *Renderable, x int) {
	r.x += x
	m.ConstrainBounds(r)
}

func (m *ExtracellularMatrix) MoveY(r *Renderable, y int) {
	r.y += y
	m.ConstrainBounds(r)
}

func (m *ExtracellularMatrix) MoveUp(r *Renderable) {
	if m.prev != nil {
		m.Detach(r)
		m.Attach(r)
		m.prev.ConstrainBounds(r)
	}
}

func (m *ExtracellularMatrix) MoveDown(r *Renderable) {
	if m.next != nil {
		m.Detach(r)
		m.Attach(r)
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
	_, hasRender := m.attached[r.id]
	if hasRender {
		delete(m.attached, r.id)
	} else if m.next != nil {
		m.next.Detach(r)
	}
	m.Unlock()
}

func (m *ExtracellularMatrix) Render(renderChan chan RenderableData) {
	for _, attached := range m.attached {
		renderChan <- m.RenderObject(attached)
	}
	close(renderChan)
}

func (m *ExtracellularMatrix) RenderObject(r *Renderable) RenderableData {
	return RenderableData{
		Id:      r.id,
		Visible: r.visible,
		X:       r.x,
		Y:       r.y,
		Z:       m.level,
	}
}

func (m *ExtracellularMatrix) GetCytokines(pt image.Point) Cytokines {
	m.cytokineMu.Lock()
	defer m.cytokineMu.Unlock()
	cytokines, hasPoint := m.cytokinesMap[pt]
	if !hasPoint {
		cytokines = make(map[CytokineType]*Cytokine)
		m.cytokinesMap[pt] = cytokines
	}
	return cytokines
}

func (m *ExtracellularMatrix) AddCytokine(r *Renderable, t CytokineType, concentration uint8) uint8 {
	_, hasRender := m.attached[r.id]
	if hasRender {
		pt := image.Point{r.x, r.y}
		cytokines := m.GetCytokines(pt)
		m.cytokineMu.Lock()
		c, hasCytokine := cytokines[t]
		if !hasCytokine {
			c = MakeCytokine(pt, t, concentration)
			cytokines[t] = c
		}
		m.cytokineMu.Unlock()
		return c.Add(concentration)
	}
	if m.next != nil {
		return m.next.AddCytokine(r, t, concentration)
	}
	return 0
}

func (m *ExtracellularMatrix) RemoveCytokine(r *Renderable, t CytokineType, concentration uint8) uint8 {
	_, hasRender := m.attached[r.id]
	if hasRender {
		pt := image.Point{r.x, r.y}
		m.cytokineMu.Lock()
		cytokines := m.GetCytokines(pt)
		c, hasCytokine := cytokines[t]
		if !hasCytokine {
			return 0
		}
		return c.Sub(concentration)
	}
	if m.next != nil {
		return m.next.RemoveCytokine(r, t, concentration)
	}
	return 0
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
	var circles []Circle
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
			circles = append(circles, Circle{boundary.Min, RandInRange(2, minX)})
			circles = append(circles, Circle{boundary.Max, RandInRange(2, minY)})
		}
	}
	return &Walls{finalBoundaries, circles, bridges}
}
