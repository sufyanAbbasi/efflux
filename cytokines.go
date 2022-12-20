package main

import (
	"image"
	"math"
	"sync"
	"time"
)

type CytokineType int

const (
	cell_damage CytokineType = iota
	antigen_found
	induce_inflammation
	induce_chemotaxis
)

func Gaussian(sigma, x float64) float64 {
	return (1 / (sigma * math.Sqrt(2*math.Pi))) * math.Exp(-0.5*math.Pow(x, 2)/math.Pow(sigma, 2))
}

type Cytokine struct {
	sync.RWMutex
	Circle
	cType           CytokineType
	concentration   uint8
	lastTick        time.Time
	tickRate        time.Duration
	dissipationRate uint8
	dissipated      bool
}

func MakeCytokine(cType CytokineType, pt image.Point, concentration uint8) *Cytokine {
	return &Cytokine{
		Circle: Circle{
			center: pt,
			radius: CYTOKINE_RADIUS,
		},
		cType:           cType,
		concentration:   concentration,
		lastTick:        time.Now(),
		tickRate:        CYTOKINE_TICK_RATE,
		dissipationRate: CYTOKINE_DISSIPATION_RATE,
		dissipated:      false,
	}
}

func (c *Cytokine) Add(x uint8) uint8 {
	c.Lock()
	defer c.Unlock()
	concentration := int(x) + int(c.concentration)
	if concentration > math.MaxUint8-1 {
		c.concentration = math.MaxUint8 - 1
	} else {
		c.concentration = uint8(concentration)
	}
	return c.concentration
}

func (c *Cytokine) Sub(x uint8) uint8 {
	c.Lock()
	defer c.Unlock()
	if x > c.concentration {
		c.concentration = 0
	} else {
		c.concentration -= x
	}
	return c.concentration
}

func (c *Cytokine) Tick() {
	c.Lock()
	defer c.Unlock()
	currTime := time.Now()
	if currTime.After(c.lastTick.Add(c.tickRate)) {
		if c.concentration <= c.dissipationRate {
			c.concentration = 0
		} else {
			c.concentration -= c.dissipationRate
		}
		c.radius += CYTOKINE_EXPANSION_RATE
		c.lastTick = currTime
	}
	if c.concentration == 0 {
		c.dissipated = true
	}
}

func (c *Cytokine) At(pt image.Point) uint8 {
	if c.InBounds(pt) {
		return uint8(math.Floor(float64(c.concentration) *
			math.Exp(-0.5*math.Pow(float64(c.Distance(pt)), 2)/math.Pow(float64(c.radius)/5, 2))))
	}
	return 0
}
