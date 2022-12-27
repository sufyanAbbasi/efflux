package main

import (
	"image"
	"math"
	"sync"
	"time"
)

type CytokineType int
type Cytokines map[CytokineType]*Cytokine

const (
	cell_damage CytokineType = iota
	antigen_present
	induce_chemotaxis
)

func Gaussian(sigma, x float64) float64 {
	return (1 / (sigma * math.Sqrt(2*math.Pi))) * math.Exp(-0.5*math.Pow(x, 2)/math.Pow(sigma, 2))
}

type Cytokine struct {
	sync.RWMutex
	Circle
	cytokine        CytokineType
	concentration   uint8
	lastTick        time.Time
	tickRate        time.Duration
	dissipationRate uint8
}

func MakeCytokine(pt image.Point, cytokine CytokineType, concentration uint8) *Cytokine {
	return &Cytokine{
		Circle: Circle{
			center: pt,
			radius: CYTOKINE_RADIUS,
		},
		cytokine:        cytokine,
		concentration:   concentration,
		lastTick:        time.Now(),
		tickRate:        CYTOKINE_TICK_RATE,
		dissipationRate: CYTOKINE_DISSIPATION_RATE,
	}
}

func (c *Cytokine) Add(x uint8) (added uint8) {
	c.Lock()
	defer c.Unlock()
	concentration := int(c.concentration) + int(x)
	if concentration >= math.MaxUint8 {
		prev := c.concentration
		c.concentration = math.MaxUint8
		added = math.MaxUint8 - prev
	} else {
		c.concentration = uint8(concentration)
		added = x
	}
	return
}

func (c *Cytokine) Sub(x uint8) (removed uint8) {
	c.Lock()
	defer c.Unlock()
	if x > c.concentration {
		removed = c.concentration
		c.concentration = 0
	} else {
		c.concentration -= x
		removed = x
	}
	return
}

func (c *Cytokine) Tick() {
	c.Lock()
	defer c.Unlock()
	currTime := time.Now()
	if currTime.After(c.lastTick.Add(c.tickRate)) {
		if c.concentration <= c.dissipationRate {
			c.concentration = 0
			c.radius = 0
		} else {
			c.concentration -= c.dissipationRate
		}
		c.radius += CYTOKINE_EXPANSION_RATE
		c.lastTick = currTime
	}
}

func (c *Cytokine) At(pt image.Point) uint8 {
	c.RLock()
	concentration := c.concentration
	c.RUnlock()
	if c.InBounds(pt) {
		return uint8(math.Floor(float64(concentration) *
			math.Exp(-0.5*math.Pow(float64(c.Distance(pt)), 2)/math.Pow(float64(c.radius)/5, 2))))
	}
	return 0
}
