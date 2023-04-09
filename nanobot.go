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
	name         string
	sessionToken uuid.UUID
	expiry       time.Time
	organ        *Node
	render       *Renderable
	stop         context.CancelFunc
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
	// Can be called multiple talls.
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
		}
	}
}

func (n *Nanobot) Stop() {
	if n.stop != nil {
		n.stop()
	}
}

func (n *Nanobot) CleanUp() {
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

func (n *Nanobot) ProcessInteraction(interaction *InteractionRequest) (bool, error) {
	n.RenewExpiry()
	n.render.visible = true
	if interaction.Type == InteractionRequest_move {
		position := interaction.Position
		if position != nil {
			n.SetTargetPoint(image.Point{int(position.X), int(position.Y)})
		}
	} else if interaction.Type == InteractionRequest_close {
		return true, nil
	}
	return false, nil
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
