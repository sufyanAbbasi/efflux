package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

type AntigenPool struct {
	viralLoads     *sync.Map
	antibodyLoads  *sync.Map
	proteinChan    chan Protein
	infectablePool *sync.Pool
}

func InitializeAntigenPool(ctx context.Context) *AntigenPool {
	antigenPool := &AntigenPool{
		viralLoads:     &sync.Map{},
		antibodyLoads:  &sync.Map{},
		proteinChan:    make(chan Protein, PROTEIN_CHAN_BUFFER),
		infectablePool: &sync.Pool{},
	}
	go antigenPool.Start(ctx)
	return antigenPool
}

func (a *AntigenPool) Start(ctx context.Context) {
	ticker := time.NewTicker(ANTIGEN_POOL_TICK_RATE)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.Tick()
		}
	}
}

func (a *AntigenPool) Tick() {
	// Pick a victim to infect or attach antibodies.
	infectable := a.infectablePool.Get()
	var antibodyLoads []*AntibodyLoad
	a.antibodyLoads.Range(func(_, a any) bool {
		antibodyLoad := a.(*AntibodyLoad)
		if infectable != nil {
			c := infectable.(CellActor)
			if antibodyLoad.ShouldAttach(c) {
				antibodyLoad.Attach(c)
			}
		}
		antibodyLoads = append(antibodyLoads, antibodyLoad)
		return true
	})
	a.viralLoads.Range(func(_, v any) bool {
		viralLoad := v.(*ViralLoad)
		viralLoad.Tick(antibodyLoads)
		if infectable != nil {
			c := infectable.(CellActor)
			if viralLoad.ShouldInfect(c) {
				viralLoad.virus.Infect(c)
			}
		}
		return true
	})
}

func (a *AntigenPool) BroadcastExistence(c CellActor) {
	a.infectablePool.Put(c)
}

func (a *AntigenPool) DepositViralLoad(v *ViralLoad) {
	viralLoad, _ := a.viralLoads.LoadOrStore(&v.virus.dna.base.D, &ViralLoad{
		virus:         v.virus,
		concentration: 0,
	})
	viralLoad.(*ViralLoad).Merge(v)
}

func (a *AntigenPool) DepositProteins(proteins []Protein) {
	for i := 0; i < PROTEIN_DEPOSIT_RATE; i++ {
		for _, protein := range proteins {
			a.proteinChan <- protein
		}
	}
}

func (a *AntigenPool) GetViralLoad() int {
	viralLoadTotal := 0
	a.viralLoads.Range(func(_, v any) bool {
		viralLoad := v.(*ViralLoad)
		viralLoad.RLock()
		if int64(viralLoadTotal)+viralLoad.concentration > math.MaxInt {
			viralLoadTotal = math.MaxInt
		} else {
			viralLoadTotal += int(viralLoad.concentration)
		}
		viralLoad.RUnlock()
		return true
	})
	return viralLoadTotal
}

func (a *AntigenPool) SampleVirusProteins(sampleRate int64) (proteins []Protein) {
	a.viralLoads.Range(func(_, v any) bool {
		viralLoad := v.(*ViralLoad)
		viralLoad.Lock()
		if viralLoad.concentration > 0 {
			proteins = append(proteins, viralLoad.virus.dna.selfProteins...)
			if viralLoad.concentration > sampleRate {
				viralLoad.concentration -= sampleRate
			} else {
				viralLoad.concentration = 0
			}
		}
		viralLoad.Unlock()
		return true
	})
	return
}

func (a *AntigenPool) SampleProteins(ctx context.Context, sampleDuration time.Duration, maxSamples int) (proteins []Protein) {
	ctx, cancel := context.WithTimeout(ctx, sampleDuration)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case protein := <-a.proteinChan:
			proteins = append(proteins, protein)
			maxSamples--
			if maxSamples <= 0 {
				return
			}
		}
	}
}

type AntibodyLoad struct {
	sync.RWMutex
	targetProtein Protein
	concentration int64
}

func (a *AntibodyLoad) ShouldAttach(antigenPresentor AntigenPresenting) bool {
	antigen := antigenPresentor.PresentAntigen()
	for _, protein := range antigen.proteins {
		if protein == a.targetProtein {
			return true
		}
	}
	return false
}

func (a *AntibodyLoad) Attach(cell CellActor) bool {
	cell.AddAntibodyLoad(&AntibodyLoad{
		targetProtein: a.targetProtein,
		concentration: 1,
	})
	a.Deplete(1)
	return false
}

func (a *AntibodyLoad) Deplete(amount int64) (depleted int64) {
	a.Lock()
	defer a.Unlock()
	if amount > a.concentration {
		depleted = a.concentration
		a.concentration = 0
	} else {
		depleted = amount
		a.concentration -= amount
	}
	return
}

func (a *AntibodyLoad) Merge(antibodyLoad *AntibodyLoad) {
	if antibodyLoad == nil || a.targetProtein != antibodyLoad.targetProtein {
		return
	}
	a.Lock()
	defer a.Unlock()
	a.concentration += antibodyLoad.concentration
}

type ViralLoad struct {
	sync.RWMutex
	virus         *Virus
	concentration int64
}

func (v *ViralLoad) ShouldInfect(cell CellActor) bool {
	v.RLock()
	defer v.RUnlock()
	if cell.ViralLoad() != nil ||
		v.concentration <= 0 ||
		cell.CellType() != v.virus.targetCellType ||
		v.virus.infectivity <= 0 {
		return false
	}
	if v.concentration >= v.virus.infectivity*10 {
		// Cap the odds at 1:10.
		return rand.Intn(10) == 0
	}
	// Generate a random number within the infectivity range. If the generated
	// number is less than or equal to the concentration, then the cell was
	// infected.
	return rand.Int63n(v.virus.infectivity) <= v.concentration
}

func (v *ViralLoad) Tick(antibodies []*AntibodyLoad) {
	v.Lock()
	defer v.Unlock()
	if v.concentration == 0 {
		return
	}
	for _, antibody := range antibodies {
		if antibody.concentration > 0 && antibody.ShouldAttach(v.virus) {
			depleted := antibody.Deplete(v.concentration)
			v.concentration -= depleted
		}
	}
}

func (v *ViralLoad) Merge(viralLoad *ViralLoad) {
	if viralLoad == nil || v.virus != viralLoad.virus {
		return
	}
	v.Lock()
	defer v.Unlock()
	v.concentration += viralLoad.concentration
}

type Virus struct {
	dna            *DNA
	targetCellType CellType
	infectivity    int64
}

func (v *Virus) String() string {
	return fmt.Sprintf("Virus (%v)", v.dna.name)
}

func (v *Virus) SampleProteins() (proteins []Protein) {
	return v.dna.selfProteins
}

func (v *Virus) PresentAntigen() *Antigen {
	return v.dna.GenerateAntigen(v.SampleProteins())
}

func (v *Virus) SetDNA(*DNA) {}

func (v *Virus) DNA() *DNA {
	return v.dna
}

func (v *Virus) Infect(c CellActor) {
	if c.CellType() == v.targetCellType && c.ViralLoad() == nil {
		c.AddViralLoad(&ViralLoad{
			virus: v,
		})
		base := c.DNA()
		proteins := append(base.selfProteins, v.dna.selfProteins...)
		c.SetDNA(&DNA{
			name:         base.name,
			base:         base.base,
			dnaType:      base.dnaType,
			selfProteins: proteins,
			makeFunction: base.makeFunction,
		})
		function := c.Function()
		if function != nil {
			function.Graft(v.dna.makeFunction(c, v.dna))
		}
		fmt.Println("Virus: ", v.dna.name, "infected", c)
	}
}
