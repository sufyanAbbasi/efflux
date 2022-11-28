package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type RenderID string

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

type World struct {
	ctx           context.Context
	bounds        image.Rectangle
	streamingChan chan chan RenderableData
	rootMatrix    *ExtracellularMatrix
}

func InitializeWorld(ctx context.Context) *World {
	world := &World{
		ctx:           ctx,
		bounds:        image.Rect(-WORLD_BOUNDS/2, -WORLD_BOUNDS/2, WORLD_BOUNDS/2, WORLD_BOUNDS/2),
		streamingChan: make(chan chan RenderableData),
		rootMatrix: &ExtracellularMatrix{
			RWMutex: sync.RWMutex{},
			level:   0,
			render: &Renderable{
				id:      RenderID(fmt.Sprintf("Matrix%v", rand.Intn(1000000))),
				visible: true,
				x:       0,
				y:       0,
			},
			attached: make(map[RenderID]*Renderable),
		},
	}
	root := world.rootMatrix
	root.world = world
	root.walls = root.GenerateWalls(WALL_LINES, WALL_BOXES)
	return world
}

func (w *World) MakeNewRenderAndAttach(idPrefix string) *Renderable {
	render := &Renderable{
		id:      RenderID(fmt.Sprintf("%v%v", idPrefix, rand.Intn(1000000))),
		visible: true,
		x:       0,
		y:       0,
	}
	w.Attach(render)
	return render
}

func (w *World) Attach(r *Renderable) {
	w.rootMatrix.Attach(r)
}

func (w *World) Detach(r *Renderable) {
	w.rootMatrix.Detach(r)
}

func (w *World) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-w.streamingChan:
			go w.rootMatrix.Render(r)
		}
	}
}

func (w *World) Stream(connection *Connection) {
	defer connection.Close()
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			buf := new(bytes.Buffer)
			err := png.Encode(buf, w.rootMatrix)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("error: %v", err)
				}
				break

			}
			img_bytes := buf.Bytes()
			err = connection.WriteMessage(websocket.BinaryMessage, img_bytes)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("error: %v", err)
				}
				break
			}
			r := make(chan RenderableData)
			w.streamingChan <- r
			for renderable := range r {
				err := connection.WriteJSON(renderable)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						fmt.Printf("error: %v", err)
					}
					break
				}
			}
		}
	}
}

type ExtracellularMatrix struct {
	sync.RWMutex
	world    *World
	level    int
	next     *ExtracellularMatrix
	prev     *ExtracellularMatrix
	walls    []image.Rectangle
	render   *Renderable
	attached map[RenderID]*Renderable
}

func (m *ExtracellularMatrix) ColorModel() color.Model {
	return color.RGBAModel
}

func (m *ExtracellularMatrix) Bounds() image.Rectangle {
	return m.world.bounds
}

func (m *ExtracellularMatrix) At(x, y int) color.Color {
	for _, b := range m.walls {
		if image.Pt(x, y).In(b) {
			return color.White
		}
	}
	return color.Black
}

func (m *ExtracellularMatrix) GenerateWalls(numLines int, numBoxesPerLine int) []image.Rectangle {
	bounds := m.world.bounds
	randInRange := func(x, y int) int {
		var min, max int
		if x < y {
			min = x
			max = y
		} else {
			min = y
			max = x
		}
		if min == max {
			return min
		}
		rand.Seed(time.Now().UnixNano())
		if min < 0 && max < 0 {
			return rand.Intn(-min+max) + min
		} else {
			return rand.Intn(max-min) + min
		}
	}
	makeRandPoint := func(rect image.Rectangle) image.Point {
		x0 := randInRange(rect.Min.X, rect.Max.X)
		y0 := randInRange(rect.Min.Y, rect.Max.Y)
		return image.Pt(x0, y0)
	}
	makeRandPointOnLine := func(p0, p1 image.Point) image.Point {
		if p0.X == p1.X {
			return image.Pt(p0.X, randInRange(p0.Y, p1.Y))
		}
		m := (p1.Y - p0.Y) / (p1.X - p0.X)
		x := randInRange(p0.X, p0.X)
		y := (p1.Y-p0.Y)*m + p0.Y
		return image.Pt(x, y)
	}
	makeRandRect := func(rect image.Rectangle) image.Rectangle {
		x0 := randInRange(rect.Min.X, rect.Max.X)
		y0 := randInRange(rect.Min.Y, rect.Max.Y)
		x1 := randInRange(rect.Min.X, rect.Max.X)
		y1 := randInRange(rect.Min.Y, rect.Max.Y)
		return image.Rect(x0, y0, x1, y1).Intersect(bounds)
	}
	var walls []image.Rectangle
	for i := 0; i < numLines; i++ {
		p0 := makeRandPoint(bounds)
		for j := 0; j < numBoxesPerLine; j++ {
			p1 := makeRandPoint(bounds)
			p2 := makeRandPointOnLine(p0, p1)
			wall := makeRandRect(bounds)
			if !p2.In(wall) {
				wall = wall.Add(wall.Min.Sub(p2)).Intersect(bounds)
			}
			walls = append(walls, wall)
			p0 = p1
		}
	}
	var final []image.Rectangle
	for _, wall := range walls {
		hasOverlap := false
		for _, checkWall := range walls {
			if wall != checkWall && checkWall.Overlaps(wall) {
				hasOverlap = true
			}
		}
		if hasOverlap {
			final = append(final, wall)
		}
	}
	return final
}

func (m *ExtracellularMatrix) ConstrainBounds(r *Renderable) {
	b := m.world.bounds
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
