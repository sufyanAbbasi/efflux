package main

import (
	"container/ring"
	"context"
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Nanobot struct {
	sync.RWMutex
	name                     string
	sessionToken             uuid.UUID
	expiry                   time.Time
	organ                    *Node
	render                   *Renderable
	stop                     context.CancelFunc
	attachToTargetCell       bool
	targetCell               RenderID
	attachedTo               *Renderable
	targetCellStatusBuffer   *ring.Ring
	attachedCellStatusBuffer *ring.Ring
}

func (n *Nanobot) RenewExpiry() {
	n.expiry = time.Now().Add(NANOBOT_SESSION_DURATION)
}

func (n *Nanobot) IsExpired() bool {
	return time.Until(n.expiry) < 0
}

func (n *Nanobot) IsIdle() bool {
	return time.Until(n.expiry) < NANOBOT_SESSION_DURATION-NANOBOT_SESSION_IDLE
}

func (n *Nanobot) Start(ctx context.Context) {
	// Start can be called multiple times.
	n.targetCellStatusBuffer = ring.New(NANOBOT_CELL_STATUS_BUFFER_SIZE)
	n.attachedCellStatusBuffer = ring.New(NANOBOT_CELL_STATUS_BUFFER_SIZE)
	if n.stop != nil {
		n.stop()
	}
	go n.Loop(ctx)
	n.Tissue().Attach(n.render)
	n.render.visible = true
}

func (n *Nanobot) Loop(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	n.stop = cancel
	defer cancel()
	ticker := time.NewTicker(NANOBOT_LOOP_DURATION)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tissue := n.Tissue()
			if tissue != nil {
				tissue.Move(n.render)
			}
			if n.targetCell != "" && (n.attachToTargetCell || n.attachedTo != nil || n.render.followId != "") {
				for _, c := range n.GetInteractions(ctx) {
					if c.Render().id == n.targetCell {
						if n.attachToTargetCell {
							n.AttachCell(c)
						}
						if n.targetCellStatusBuffer.Len() > 0 {
							n.targetCellStatusBuffer = n.targetCellStatusBuffer.Next()
						}
						n.targetCellStatusBuffer.Value = c.GetCellStatus()
					}
					if n.attachedTo != nil && c.Render().id == n.attachedTo.id {
						if n.attachedCellStatusBuffer.Len() > 0 {
							n.attachedCellStatusBuffer = n.attachedCellStatusBuffer.Next()
						}
						n.attachedCellStatusBuffer.Value = c.GetCellStatus()
					}
				}
			}
		}
	}
}

func (n *Nanobot) Stop() {
	if n.stop != nil {
		n.stop()
	}
}

func (n *Nanobot) CleanUp() {
	n.DetachCell()
	n.render.followId = ""
	n.render.visible = false
	if n.organ != nil && n.organ.tissue != nil {
		n.organ.tissue.Detach(n.render)
	}
	n.Stop()
	if n.Verbose() {
		fmt.Println("Despawned:", n, "in", n.organ)
	}
	n.organ = nil
}

func (n *Nanobot) String() string {
	return fmt.Sprintf("Nanobot #%v", n.name)
}

func (n *Nanobot) Verbose() bool {
	return n.organ != nil && n.organ.verbose
}

func (n *Nanobot) Organ() *Node {
	return n.organ
}

func (n *Nanobot) SetOrgan(node *Node) {
	n.organ = node
}

func (n *Nanobot) Tissue() *Tissue {
	if n.organ == nil ||
		n.organ.tissue == nil ||
		n.organ.tissue.rootMatrix == nil {
		return nil
	}
	return n.organ.tissue
}

func (n *Nanobot) Render() *Renderable {
	return n.render
}

func (n *Nanobot) Position() image.Point {
	return n.render.position
}

func (n *Nanobot) LastPositions() *ring.Ring {
	return n.render.lastPositions
}

func (n *Nanobot) SetTargetPoint(pt image.Point) {
	if n.render == nil {
		return
	}
	n.render.targetX = pt.X
	n.render.targetY = pt.Y
}

func (n *Nanobot) GetInteractions(ctx context.Context) (interactions []CellActor) {
	tissue := n.Tissue()
	if tissue == nil {
		return
	}
	interactions = tissue.GetInteractions(ctx, n.render)
	return
}

func (n *Nanobot) ProcessInteraction(request *InteractionRequest, response *InteractionResponse) (bool, error) {
	n.RenewExpiry()
	n.render.visible = true
	var err error
	toClose := false
	fmt.Println("got request", request.Type, request.TargetCell)
	switch request.Type {
	case InteractionType_close:
		toClose = true
		n.DetachCell()
	case InteractionType_move_to:
		n.render.followId = ""
		position := request.Position
		if position != nil {
			n.SetTargetPoint(image.Point{int(position.X), int(position.Y)})
		}
	case InteractionType_info:
		if request.TargetCell == "" {
			err = fmt.Errorf("got info interaction but had empty target ID: %v", request.TargetCell)
		} else {
			target := RenderID(request.TargetCell)
			n.targetCell = target
		}
	case InteractionType_follow:
		if request.TargetCell == "" {
			err = fmt.Errorf("got follow interaction but had empty target ID: %v", request.TargetCell)
		} else {
			n.targetCell = RenderID(request.TargetCell)
		}
		n.render.followId = n.targetCell
	case InteractionType_attach:
		if request.TargetCell == "" {
			err = fmt.Errorf("got attach interaction but had empty target ID: %v", request.TargetCell)
		} else {
			n.targetCell = RenderID(request.TargetCell)
			n.attachToTargetCell = true
		}
	case InteractionType_detach:
		n.DetachCell()
	case InteractionType_drop_cytokine:
		fmt.Println("got cytokine", request.CytokineType)
		if request.CytokineType == CytokineType_unknown {
			err = fmt.Errorf("got drop cytokine interaction but had empty cytokine: %v", request.CytokineType)
		} else {
			n.DropCytokine(request.CytokineType)
		}
	}
	if err != nil {
		fmt.Println("", err)
		response.Status = InteractionResponse_failure
		response.ErrorMessage = fmt.Sprint(err)
	} else {
		response.Status = InteractionResponse_success
		if n.attachedTo != nil {
			response.AttachedTo = string(n.attachedTo.id)
		}
		if n.targetCellStatusBuffer != nil {
			status, ok := n.targetCellStatusBuffer.Value.(*CellStatus)
			if ok {
				response.TargetCellStatus = status
				n.targetCellStatusBuffer.Value = nil
			}

		}
		if n.attachedCellStatusBuffer != nil {
			status, ok := n.attachedCellStatusBuffer.Value.(*CellStatus)
			if ok {
				response.AttachedCellStatus = status
				n.attachedCellStatusBuffer.Value = nil
			}

		}
	}
	return toClose, err
}

func (n *Nanobot) AttachCell(c CellActor) {
	n.DetachCell()
	n.render.followId = ""
	c.Render().followId = n.render.id
	n.attachedTo = c.Render()
	fmt.Println(n.render.id, "attached to", c.Render().id)
}

func (n *Nanobot) DetachCell() {
	n.targetCell = ""
	n.attachToTargetCell = false
	if n.attachedTo != nil {
		fmt.Println(n.render.id, "detached from", n.attachedTo.id)
		n.attachedTo.followId = ""
		n.attachedTo = nil
	}
}

func (n *Nanobot) DropCytokine(t CytokineType) {
	tissue := n.Tissue()
	if tissue != nil {
		tissue.AddCytokine(n.render, t, NANOBOT_CYTOKINE_CONCENTRATION)
	}
}

type NanobotManager struct {
	nanobots *sync.Map
}

func InitializeNanobotManager(ctx context.Context) *NanobotManager {
	return &NanobotManager{
		nanobots: &sync.Map{},
	}
}

func (m *NanobotManager) Start(ctx context.Context) {
	m.GarbageCollect(ctx)
}

func (m *NanobotManager) GarbageCollect(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ticker := time.NewTicker(NANOBOT_GC_CLOCK_RATE)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var toDelete []uuid.UUID
			m.nanobots.Range(func(key, value any) bool {
				nanobot := value.(*Nanobot)
				if nanobot.IsExpired() {
					toDelete = append(toDelete, key.(uuid.UUID))
					nanobot.CleanUp()
				} else if nanobot.IsIdle() {
					nanobot.render.visible = false
				}
				return true
			})
			for _, id := range toDelete {
				m.nanobots.Delete(id)
			}
		}
	}
}
