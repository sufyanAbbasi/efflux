package main

import (
	"context"
	"sync"
)

type ResourceBlob struct {
	o2       int
	glucose  int
	vitamins int
}

func offset(val int) (offset int) {
	if val%2 == 0 {
		offset = 1
	}
	return
}

func (r *ResourceBlob) Consume(need *ResourceBlob) {
	if r.o2 >= need.o2 {
		r.o2 -= need.o2
		need.o2 = 0
	} else {
		r.o2 = 0
		need.o2 -= r.o2
	}
	if r.vitamins >= need.vitamins {
		r.vitamins -= need.vitamins
		need.vitamins = 0
	} else {
		r.vitamins = 0
		need.vitamins -= r.vitamins
	}
	if r.glucose >= need.glucose {
		r.glucose -= need.glucose
		need.glucose = 0
	} else {
		r.glucose = 0
		need.glucose -= r.glucose
	}
}

func (r *ResourceBlob) Add(resource *ResourceBlob) {
	r.o2 += resource.o2
	r.vitamins += resource.vitamins
	r.glucose += resource.glucose
}

func (r *ResourceBlob) Split() *ResourceBlob {
	keep := &ResourceBlob{
		o2:       0,
		glucose:  0,
		vitamins: 0,
	}
	if r.o2 > 1 {
		offset := offset(r.o2)
		r.o2 /= 2
		keep.o2 += r.o2 + offset
	}
	if r.glucose > 1 {
		offset := offset(r.glucose)
		r.glucose /= 2
		keep.glucose += r.glucose + offset
	}
	if r.vitamins > 1 {
		offset := offset(r.vitamins)
		r.vitamins /= 2
		keep.vitamins += r.vitamins + offset
	}
	return keep
}

type WasteBlob struct {
	co2        int
	creatinine int
}

func (w *WasteBlob) Add(waste *WasteBlob) {
	w.co2 += waste.co2
	w.creatinine += waste.creatinine
}

func (w *WasteBlob) Split() *WasteBlob {
	keep := &WasteBlob{
		co2:        0,
		creatinine: 0,
	}
	if w.co2 > 1 {
		offset := offset(w.co2)
		w.co2 /= 2
		keep.co2 += w.co2 + offset
	}
	if w.creatinine > 1 {
		offset := offset(w.creatinine)
		w.creatinine /= 2
		keep.creatinine += w.creatinine + offset
	}
	return keep
}

type LigandBlob struct {
	growth       int
	hunger       int
	asphyxia     int
	inflammation int
}

func (l *LigandBlob) Add(ligand *LigandBlob) {
	l.growth += ligand.growth
	l.hunger += ligand.hunger
	l.asphyxia += ligand.asphyxia
	l.inflammation += ligand.inflammation
	if l.inflammation > LIGAND_INFLAMMATION_MAX {
		l.inflammation = LIGAND_INFLAMMATION_MAX
	}
}

func (l *LigandBlob) Split() *LigandBlob {
	keep := &LigandBlob{
		growth:       0,
		hunger:       0,
		asphyxia:     0,
		inflammation: 0,
	}
	if l.growth > 1 {
		offset := offset(l.growth)
		l.growth /= 2
		keep.growth += l.growth + offset
	}
	if l.hunger > 1 {
		offset := offset(l.hunger)
		l.hunger /= 2
		keep.hunger += l.hunger + offset
	}
	if l.asphyxia > 1 {
		offset := offset(l.asphyxia)
		l.asphyxia /= 2
		keep.asphyxia += l.asphyxia + offset
	}
	if l.inflammation > 1 {
		offset := offset(l.inflammation)
		l.inflammation /= 2
		keep.inflammation += l.inflammation + offset
	}
	return keep
}

type HormoneBlob struct {
	granulocyte_csf int // Produces Myeloblast.
	macrophage_csf  int // Produces Monocyte.
	interleukin_3   int // Produces Lymphoblast.
	interleukin_2   int // Induces TCell mitosis.
}

func (h *HormoneBlob) Add(hormone *HormoneBlob) {
	h.granulocyte_csf += hormone.granulocyte_csf
	h.macrophage_csf += hormone.macrophage_csf
	h.interleukin_3 += hormone.interleukin_3
	h.interleukin_2 += hormone.interleukin_2
}

func (h *HormoneBlob) Split() *HormoneBlob {
	keep := &HormoneBlob{
		granulocyte_csf: 0,
		macrophage_csf:  0,
		interleukin_3:   0,
		interleukin_2:   0,
	}
	if h.granulocyte_csf > 1 {
		offset := offset(h.granulocyte_csf)
		h.granulocyte_csf /= 2
		keep.granulocyte_csf += h.granulocyte_csf + offset
	}
	if h.macrophage_csf > 1 {
		offset := offset(h.macrophage_csf)
		h.macrophage_csf /= 2
		keep.macrophage_csf += h.macrophage_csf + offset
	}
	if h.interleukin_3 > 1 {
		offset := offset(h.interleukin_3)
		h.interleukin_3 /= 2
		keep.interleukin_3 += h.interleukin_3 + offset
	}
	if h.interleukin_2 > 1 {
		offset := offset(h.interleukin_2)
		h.interleukin_2 /= 2
		keep.interleukin_2 += h.interleukin_2 + offset
	}
	return keep
}

type ResourcePool struct {
	sync.RWMutex
	resources    *ResourceBlob
	resourceChan chan *ResourceBlob
	wantChan     chan struct{}
}

func (p *ResourcePool) Check() *ResourceBlob {
	return &ResourceBlob{
		o2:       p.resources.o2,
		glucose:  p.resources.glucose,
		vitamins: p.resources.vitamins,
	}
}

func (p *ResourcePool) Get(ctx context.Context) *ResourceBlob {
	ctx, cancel := context.WithTimeout(ctx, TIMEOUT_SEC)
	defer cancel()
	select {
	case <-ctx.Done():
		return &ResourceBlob{}
	case p.wantChan <- struct{}{}:
		select {
		case <-ctx.Done():
			return &ResourceBlob{}
		case blob := <-p.resourceChan:
			return blob
		}
	}
}

func (p *ResourcePool) Put(r *ResourceBlob) {
	p.Lock()
	p.resources.Add(r)
	p.Unlock()
}

func (p *ResourcePool) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			panic("ResourcePool canceled")
		case r := <-p.resourceChan:
			p.resources.Add(r)
		case <-p.wantChan:
			p.resourceChan <- p.resources.Split()
		}
	}
}

type WastePool struct {
	sync.RWMutex
	wastes    *WasteBlob
	wasteChan chan *WasteBlob
	wantChan  chan struct{}
}

func (p *WastePool) Get(ctx context.Context) *WasteBlob {
	ctx, cancel := context.WithTimeout(ctx, TIMEOUT_SEC)
	defer cancel()
	select {
	case <-ctx.Done():
		return &WasteBlob{}
	case p.wantChan <- struct{}{}:
		select {
		case <-ctx.Done():
			return &WasteBlob{}
		case blob := <-p.wasteChan:
			return blob
		}
	}
}

func (p *WastePool) Put(r *WasteBlob) {
	p.Lock()
	p.wastes.Add(r)
	p.Unlock()
}

func (p *WastePool) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			panic("WastePool canceled")
		case r := <-p.wasteChan:
			p.wastes.Add(r)
		case <-p.wantChan:
			p.wasteChan <- p.wastes.Split()
		}
	}
}

type LigandPool struct {
	sync.RWMutex
	ligands    *LigandBlob
	ligandChan chan *LigandBlob
	wantChan   chan struct{}
}

func (p *LigandPool) Get(ctx context.Context) *LigandBlob {
	ctx, cancel := context.WithTimeout(ctx, TIMEOUT_SEC)
	defer cancel()
	select {
	case <-ctx.Done():
		return &LigandBlob{}
	case p.wantChan <- struct{}{}:
		select {
		case <-ctx.Done():
			return &LigandBlob{}
		case blob := <-p.ligandChan:
			return blob
		}
	}
}

func (p *LigandPool) Put(r *LigandBlob) {
	p.Lock()
	p.ligands.Add(r)
	p.Unlock()
}

func (p *LigandPool) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			panic("LigandPool canceled")
		case r := <-p.ligandChan:
			p.ligands.Add(r)
		default:
			<-p.wantChan
			p.ligandChan <- p.ligands.Split()
		}
	}
}

type HormonePool struct {
	sync.RWMutex
	hormones    *HormoneBlob
	hormoneChan chan *HormoneBlob
	wantChan    chan struct{}
}

func (p *HormonePool) Get(ctx context.Context) *HormoneBlob {
	ctx, cancel := context.WithTimeout(ctx, TIMEOUT_SEC)
	defer cancel()
	select {
	case <-ctx.Done():
		return &HormoneBlob{}
	case p.wantChan <- struct{}{}:
		select {
		case <-ctx.Done():
			return &HormoneBlob{}
		case blob := <-p.hormoneChan:
			return blob
		}
	}
}

func (p *HormonePool) Put(r *HormoneBlob) {
	p.Lock()
	p.hormones.Add(r)
	p.Unlock()
}

func (p *HormonePool) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			panic("HormonePool")
		case r := <-p.hormoneChan:
			p.hormones.Add(r)
		default:
			<-p.wantChan
			p.hormoneChan <- p.hormones.Split()
		}
	}
}

type MaterialPool struct {
	resourcePool *ResourcePool
	wastePool    *WastePool
	ligandPool   *LigandPool
	hormonePool  *HormonePool
}

func InitializeMaterialPool(ctx context.Context) *MaterialPool {
	m := &MaterialPool{
		resourcePool: &ResourcePool{
			resources: &ResourceBlob{
				o2:       SEED_O2,
				glucose:  SEED_GLUCOSE,
				vitamins: SEED_VITAMINS,
			},
			resourceChan: make(chan *ResourceBlob, POOL_SIZE),
			wantChan:     make(chan struct{}, POOL_SIZE),
		},
		wastePool: &WastePool{
			wastes: &WasteBlob{
				co2:        0,
				creatinine: 0,
			},
			wasteChan: make(chan *WasteBlob, POOL_SIZE),
			wantChan:  make(chan struct{}, POOL_SIZE),
		},
		ligandPool: &LigandPool{
			ligands: &LigandBlob{
				growth: SEED_GROWTH,
			},
			ligandChan: make(chan *LigandBlob, POOL_SIZE),
			wantChan:   make(chan struct{}, POOL_SIZE),
		},
		hormonePool: &HormonePool{
			hormones: &HormoneBlob{
				granulocyte_csf: SEED_GRANULOCYTE_COLONY_STIMULATING_FACTOR,
				macrophage_csf:  SEED_MACROPHAGE_COLONY_STIMULATING_FACTOR,
				interleukin_3:   SEED_INTERLEUKIN_3,
				interleukin_2:   SEED_INTERLEUKIN_2,
			},
			hormoneChan: make(chan *HormoneBlob, POOL_SIZE),
			wantChan:    make(chan struct{}, POOL_SIZE),
		},
	}
	go m.resourcePool.Start(ctx)
	go m.wastePool.Start(ctx)
	go m.ligandPool.Start(ctx)
	go m.hormonePool.Start(ctx)
	return m
}

func (m *MaterialPool) GetResource(ctx context.Context) *ResourceBlob {
	return m.resourcePool.Get(ctx)
}

func (m *MaterialPool) SplitResource(ctx context.Context) *ResourceBlob {
	blob := m.resourcePool.Get(ctx)
	m.PutResource(blob.Split())
	return blob
}

func (m *MaterialPool) PutResource(r *ResourceBlob) {
	m.resourcePool.Put(r)
}

func (m *MaterialPool) GetWaste(ctx context.Context) *WasteBlob {
	return m.wastePool.Get(ctx)
}

func (m *MaterialPool) SplitWaste(ctx context.Context) *WasteBlob {
	blob := m.wastePool.Get(ctx)
	m.PutWaste(blob.Split())
	return blob
}

func (m *MaterialPool) PutWaste(w *WasteBlob) {
	m.wastePool.Put(w)
}

func (m *MaterialPool) GetLigand(ctx context.Context) *LigandBlob {
	return m.ligandPool.Get(ctx)
}

func (m *MaterialPool) PutLigand(l *LigandBlob) {
	m.ligandPool.Put(l)
}

func (m *MaterialPool) GetHormone(ctx context.Context) *HormoneBlob {
	return m.hormonePool.Get(ctx)
}

func (m *MaterialPool) SplitHormone(ctx context.Context) *HormoneBlob {
	blob := m.hormonePool.Get(ctx)
	m.PutHormone(blob.Split())
	return blob
}

func (m *MaterialPool) PutHormone(c *HormoneBlob) {
	m.hormonePool.Put(c)
}
