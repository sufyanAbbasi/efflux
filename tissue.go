package main

import (
	"container/ring"
	"context"
	"fmt"
	"sync"
	"time"

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
	Id       RenderID `json:"id"`
	Visible  bool     `json:"visible"`
	X        float32  `json:"x"`
	Y        float32  `json:"y"`
	Z        float32  `json:"z"`
	Color    uint32   `json:"color"`
	Geometry string   `json:"geometry"`
}

type RenderCache struct {
	sync.RWMutex
	changedX, changedY, changedZ,
	changedVisible, changedColor, changedGeometry bool
	connectionBuffer *ring.Ring
	TTL              time.Duration
	lastResetTime    time.Time
}

func (r *RenderCache) Reset() {
	r.changedX = false
	r.changedY = false
	r.changedZ = false
	r.changedVisible = false
	r.changedColor = false
	r.changedGeometry = false
}

func (r *RenderCache) Report(conn *websocket.Conn) bool {
	n := r.connectionBuffer.Len()
	found := false
	curr := r.connectionBuffer
	for i := 0; i < n; i++ {
		if conn == curr.Value.(*websocket.Conn) {
			found = true
		}
		curr = curr.Prev()
	}
	r.Lock()
	if !found {
		r.connectionBuffer.Value = conn
		r.connectionBuffer = r.connectionBuffer.Next()
	}
	r.Unlock()
	r.Lock()
	now := time.Now()
	if r.lastResetTime.Add(r.TTL).After(now) {
		r.Reset()
		r.lastResetTime = now
	}
	r.Unlock()
	return found
}

type Render struct {
	visible     bool
	x, y, z     float32
	color       uint32
	geometry    string
	renderCache *RenderCache
}

func (r *Render) MoveX(x float32) {
	if x == 0 {
		return
	}
	r.x += x
	r.renderCache.Lock()
	r.renderCache.changedX = true
	r.renderCache.Unlock()
}

func (r *Render) MoveY(y float32) {
	if y == 0 {
		return
	}
	r.y += y
	r.renderCache.Lock()
	r.renderCache.changedY = true
	r.renderCache.Unlock()
}

func (r *Render) MoveZ(z float32) {
	if z == 0 {
		return
	}
	r.z += z
	r.renderCache.Lock()
	r.renderCache.changedZ = true
	r.renderCache.Unlock()
}

func (r *Render) ChangeColor(color uint32) {
	r.color = color
	r.renderCache.Lock()
	r.renderCache.changedColor = true
	r.renderCache.Unlock()
}

func (r *Render) ChangeGeometry(geometry string) {
	r.geometry = geometry
	r.renderCache.Lock()
	r.renderCache.changedGeometry = true
	r.renderCache.Unlock()
}

func (r *Render) SetVisible(visible bool) {
	r.visible = visible
	r.renderCache.Lock()
	r.renderCache.changedVisible = true
	r.renderCache.Unlock()
}

func (r *Render) RenderDiff(renderId RenderID, conn *websocket.Conn) RenderData {
	renderData := RenderData{
		Id:      renderId,
		Visible: r.visible,
	}
	fullRender := r.renderCache.Report(conn)
	if fullRender || r.renderCache.changedX {
		renderData.X = r.x
	}
	if fullRender || r.renderCache.changedY {
		renderData.Y = r.y
	}
	if fullRender || r.renderCache.changedZ {
		renderData.Z = r.z
	}
	if fullRender || r.renderCache.changedColor {
		renderData.Color = r.color
	}
	if fullRender || r.renderCache.changedGeometry {
		renderData.Geometry = r.geometry
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
	renderChan    chan chan *Renderable
	streamingChan chan struct{}
	rootMatrix    *ExtracellularMatrix
}

func (w *World) Attach(r *Renderable) func() {
	return w.rootMatrix.Attach(r)
}

func (w *World) Start(ctx context.Context) {
	for {
		if w.rootMatrix != nil {
			select {
			case <-ctx.Done():
				return
			case <-w.streamingChan:
				renderChan := make(chan *Renderable)
				go w.rootMatrix.Render(renderChan)
				w.renderChan <- renderChan
			}
		} else {
			return
		}
	}
}

func (w *World) Stream(connection *websocket.Conn) {
	defer connection.Close()
	for {
		select {
		case <-w.ctx.Done():
			return
		case w.streamingChan <- struct{}{}:
		case renderables := <-w.renderChan:
			for renderable := range renderables {
				render := renderable.render
				render.ConstrainBounds(w.bounds)
				err := websocket.JSON.Send(connection, render.RenderDiff(renderable.id, connection))
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
