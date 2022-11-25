package main

type ResourceBlob struct {
	o2       int
	glucose  int
	vitamins int
}

type ResourceBlobData struct {
	O2       int
	Glucose  int
	Vitamins int
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
		r.o2 /= 2
		keep.o2 += r.o2
	}
	if r.glucose > 1 {
		r.glucose /= 2
		keep.glucose += r.glucose
	}
	if r.vitamins > 1 {
		r.vitamins /= 2
		keep.vitamins += r.vitamins
	}
	return keep
}

type WasteBlob struct {
	co2        int
	creatinine int
}

type WasteBlobData struct {
	CO2        int
	Creatinine int
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
		w.co2 /= 2
		keep.co2 += w.co2
	}
	if w.creatinine > 1 {
		w.creatinine /= 2
		keep.creatinine += w.creatinine
	}
	return keep
}

type LigandBlob struct {
	growth   int
	hunger   int
	asphyxia int
}

func (l *LigandBlob) Add(ligand *LigandBlob) {
	l.growth += ligand.growth
	l.hunger += ligand.hunger
	l.asphyxia += ligand.asphyxia
}

func (l *LigandBlob) Split() *LigandBlob {
	keep := &LigandBlob{
		growth:   0,
		hunger:   0,
		asphyxia: 0,
	}
	if l.growth > 1 {
		l.growth /= 2
		keep.growth += l.growth
	}
	if l.hunger > 1 {
		l.hunger /= 2
		keep.hunger += l.hunger
	}
	if l.asphyxia > 1 {
		l.asphyxia /= 2
		keep.asphyxia += l.asphyxia
	}
	return keep
}

type ResourcePool struct {
	resources    *ResourceBlob
	resourceChan chan *ResourceBlob
	wantChan     chan struct{}
}

func (p *ResourcePool) Get() *ResourceBlob {
	p.wantChan <- struct{}{}
	return <-p.resourceChan
}

func (p *ResourcePool) Put(r *ResourceBlob) {
	p.resourceChan <- r
}

func (p *ResourcePool) Start() {
	for {
		select {
		case r := <-p.resourceChan:
			p.resources.Add(r)
		case <-p.wantChan:
			p.resourceChan <- p.resources.Split()
		}
	}
}

type WastePool struct {
	wastes    *WasteBlob
	wasteChan chan *WasteBlob
	wantChan  chan struct{}
}

func (p *WastePool) Get() *WasteBlob {
	p.wantChan <- struct{}{}
	return <-p.wasteChan
}

func (p *WastePool) Put(r *WasteBlob) {
	p.wasteChan <- r
}

func (p *WastePool) Start() {
	for {
		select {
		case r := <-p.wasteChan:
			p.wastes.Add(r)
		case <-p.wantChan:
			p.wasteChan <- p.wastes.Split()
		}
	}
}

type LigandPool struct {
	ligands    *LigandBlob
	ligandChan chan *LigandBlob
	wantChan   chan struct{}
}

func (p *LigandPool) Get() *LigandBlob {
	p.wantChan <- struct{}{}
	return <-p.ligandChan
}

func (p *LigandPool) Put(r *LigandBlob) {
	p.ligandChan <- r
}

func (p *LigandPool) Start() {
	for {
		select {
		case r := <-p.ligandChan:
			p.ligands.Add(r)
		default:
			<-p.wantChan
			p.ligandChan <- p.ligands.Split()
		}
	}
}

type MaterialPool struct {
	resourcePool *ResourcePool
	wastePool    *WastePool
	ligandPool   *LigandPool
}

func InitializeMaterialPool() *MaterialPool {
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
	}
	go m.resourcePool.Start()
	go m.wastePool.Start()
	go m.ligandPool.Start()
	return m
}

func (m *MaterialPool) GetResource() *ResourceBlob {
	return m.resourcePool.Get()
}

func (m *MaterialPool) SplitResource() *ResourceBlob {
	blob := m.resourcePool.Get()
	m.PutResource(blob.Split())
	return blob
}

func (m *MaterialPool) PutResource(r *ResourceBlob) {
	m.resourcePool.Put(r)
}

func (m *MaterialPool) GetWaste() *WasteBlob {
	return m.wastePool.Get()
}

func (m *MaterialPool) SplitWaste() *WasteBlob {
	blob := m.wastePool.Get()
	m.PutWaste(blob.Split())
	return blob
}

func (m *MaterialPool) PutWaste(w *WasteBlob) {
	m.wastePool.Put(w)
}

func (m *MaterialPool) GetLigand() *LigandBlob {
	return m.ligandPool.Get()
}

func (m *MaterialPool) PutLigand(l *LigandBlob) {
	m.ligandPool.Put(l)
}
