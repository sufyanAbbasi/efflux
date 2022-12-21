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

type Cytokines struct {
	sync.RWMutex
	Circle
	concentrations  map[CytokineType]uint8
	lastTick        time.Time
	tickRate        time.Duration
	dissipationRate uint8
}

func MakeCytokine(pt image.Point, concentrations map[CytokineType]uint8) *Cytokines {
	return &Cytokines{
		Circle: Circle{
			center: pt,
			radius: CYTOKINE_RADIUS,
		},
		concentrations:  concentrations,
		lastTick:        time.Now(),
		tickRate:        CYTOKINE_TICK_RATE,
		dissipationRate: CYTOKINE_DISSIPATION_RATE,
	}
}

func (c *Cytokines) Add(t CytokineType, x uint8) uint8 {
	c.Lock()
	defer c.Unlock()
	concentration := int(c.concentrations[t]) + int(x)
	if concentration > math.MaxUint8-1 {
		c.concentrations[t] = math.MaxUint8 - 1
	} else {
		c.concentrations[t] = uint8(concentration)
	}
	return c.concentrations[t]
}

func (c *Cytokines) Sub(t CytokineType, x uint8) uint8 {
	c.Lock()
	defer c.Unlock()
	if x > c.concentrations[t] {
		c.concentrations[t] = 0
	} else {
		c.concentrations[t] -= x
	}
	return c.concentrations[t]
}

func (c *Cytokines) Tick() {
	c.Lock()
	defer c.Unlock()
	currTime := time.Now()
	if currTime.After(c.lastTick.Add(c.tickRate)) {
		for t, concentration := range c.concentrations {
			if concentration <= c.dissipationRate {
				c.concentrations[t] = 0
				c.radius = 0
			} else {
				c.concentrations[t] -= c.dissipationRate
			}
		}
		c.radius += CYTOKINE_EXPANSION_RATE
		c.lastTick = currTime
	}
}

func (c *Cytokines) At(t CytokineType, pt image.Point) uint8 {
	c.RLock()
	concentration := c.concentrations[t]
	c.RUnlock()
	if c.InBounds(pt) {
		return uint8(math.Floor(float64(concentration) *
			math.Exp(-0.5*math.Pow(float64(c.Distance(pt)), 2)/math.Pow(float64(c.radius)/5, 2))))
	}
	return 0
}
