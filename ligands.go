package main

import "sync"

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
}

func (r *ResourceBlob) Add(resource *ResourceBlob) {
	r.o2 += resource.o2
	r.vitamins += resource.vitamins
}

type WasteBlob struct {
	co2      int
	toxins   int
	antigens []Protein
}

type WasteBlobData struct {
	CO2      int
	Toxins   int
	Antigens []Protein
}

func (w *WasteBlob) Add(waste *WasteBlob) {
	w.co2 += waste.co2
	w.toxins += waste.toxins
	w.antigens = append(w.antigens, waste.antigens...)
}

type LigandBlob struct {
	growth int
}

type MaterialPool struct {
	resourcePool      sync.Pool
	localResourcePool sync.Pool
	wastePool         sync.Pool
	ligandPool        sync.Pool
}

func InitializeMaterialPool() *MaterialPool {
	return &MaterialPool{
		resourcePool: sync.Pool{
			New: func() interface{} {
				return new(ResourceBlob)
			},
		},
		localResourcePool: sync.Pool{
			New: func() interface{} {
				return new(ResourceBlob)
			},
		},
		wastePool: sync.Pool{
			New: func() interface{} {
				return new(WasteBlob)
			},
		},
		ligandPool: sync.Pool{
			New: func() interface{} {
				return new(LigandBlob)
			},
		},
	}
}

func (m *MaterialPool) GetResource() *ResourceBlob {
	return m.resourcePool.Get().(*ResourceBlob)
}

func (m *MaterialPool) PutResource(r *ResourceBlob) {
	m.resourcePool.Put(r)
}

func (m *MaterialPool) GetLocalResource() *ResourceBlob {
	return m.localResourcePool.Get().(*ResourceBlob)
}

func (m *MaterialPool) PutLocalResource(r *ResourceBlob) {
	m.localResourcePool.Put(r)
}

func (m *MaterialPool) GetWaste() *WasteBlob {
	return m.wastePool.Get().(*WasteBlob)
}

func (m *MaterialPool) PutWaste(w *WasteBlob) {
	m.wastePool.Put(w)
}

func (m *MaterialPool) GetLigand() *LigandBlob {
	return m.ligandPool.Get().(*LigandBlob)
}

func (m *MaterialPool) PutLigand(l *LigandBlob) {
	m.ligandPool.Put(l)
}
