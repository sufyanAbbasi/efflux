package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"sync"

	"golang.org/x/net/websocket"
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
		bounds:        image.Rect(-WORLD_BOUNDS, -WORLD_BOUNDS, WORLD_BOUNDS, WORLD_BOUNDS),
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
	world.rootMatrix.world = world
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

func (w *World) Stream(connection *websocket.Conn) {
	defer connection.Close()
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			r := make(chan RenderableData)
			w.streamingChan <- r
			for renderable := range r {
				err := websocket.JSON.Send(connection, renderable)
				if err != nil {
					fmt.Println(err)
					return
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
	if x%10 == 0 || y%10 == 0 {
		return color.Black
	}
	return color.White
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
