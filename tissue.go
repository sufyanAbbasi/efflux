package main

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/net/websocket"
)

type Bounds struct {
	minX, minY, minZ float32
	maxX, maxY, maxZ float32
}

type RenderID string
type Renderable struct {
	id     RenderID
	render *Render
}

type RenderData struct {
	Id      RenderID `json:"id"`
	Visible bool     `json:"visible"`
	X       float32  `json:"x"`
	Y       float32  `json:"y"`
	Z       float32  `json:"z"`
}

type Render struct {
	visible bool
	x, y, z float32
}

func (r *Render) MoveX(x float32) {
	if x == 0 {
		return
	}
	r.x += x
}

func (r *Render) MoveY(y float32) {
	if y == 0 {
		return
	}
	r.y += y
}

func (r *Render) MoveZ(z float32) {
	if z == 0 {
		return
	}
	r.z += z
}

func (r *Render) SetVisible(visible bool) {
	r.visible = visible
}

func (r *Render) RenderData(renderId RenderID) RenderData {
	renderData := RenderData{
		Id:      renderId,
		Visible: r.visible,
		X:       r.x,
		Y:       r.y,
		Z:       r.z,
	}
	return renderData
}

func (r *Render) ConstrainBounds(b *Bounds) {
	if r.x > b.maxX {
		r.x = b.maxX
	}
	if r.x < b.minX {
		r.x = b.minX
	}
	if r.y > b.maxY {
		r.y = b.maxY
	}
	if r.y < b.minY {
		r.y = b.minY
	}
	if r.z > b.maxZ {
		r.z = b.maxZ
	}
	if r.z < b.minZ {
		r.z = b.minZ
	}
}

type World struct {
	ctx           context.Context
	bounds        *Bounds
	streamingChan chan chan *Renderable
	rootMatrix    *ExtracellularMatrix
}

func (w *World) Attach(r *Renderable) func() {
	return w.rootMatrix.Attach(r)
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
			r := make(chan *Renderable)
			w.streamingChan <- r
			for renderable := range r {
				render := renderable.render
				render.ConstrainBounds(w.bounds)
				err := websocket.JSON.Send(connection, render.RenderData(renderable.id))
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
	attached []*Renderable
}

func (m *ExtracellularMatrix) Render(renderChan chan *Renderable) {
	for _, attached := range m.attached {
		renderChan <- attached
	}
	close(renderChan)
}

func (m *ExtracellularMatrix) Attach(r *Renderable) func() {
	m.Lock()
	defer m.Unlock()
	for i, attached := range m.attached {
		if attached != nil {
			m.attached[i] = r
			return func() {
				m.Detach(i)
			}
		}
	}
	m.attached = append(m.attached, r)
	return func() {
		m.Detach(len(m.attached) - 1)
	}
}

func (m *ExtracellularMatrix) Detach(i int) {
	m.Lock()
	if i < len(m.attached)-1 {
		m.attached[i] = nil
	}
	m.Unlock()
}
