package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"log"
)

var HUMAN_DNA = elliptic.P521()
var BACTERIA_DNA = elliptic.P384()
var VIRUS_RNA = elliptic.P224()

type DNAType elliptic.Curve
type DNA struct {
	name         string
	base         *ecdsa.PrivateKey
	dnaType      DNAType
	selfProteins map[Protein]bool
	makeFunction func(c CellActor) *StateDiagram
}
type MHC_I *ecdsa.PublicKey
type Protein uint16
type AntigenSignature []byte

type Antigen struct {
	proteins  []Protein
	signature AntigenSignature
}

func MakeDNA(dnaType DNAType, name string) *DNA {
	// Caution: slow!
	privateKey, err := ecdsa.GenerateKey(dnaType, rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	dna := &DNA{
		name:    name,
		base:    privateKey,
		dnaType: dnaType,
	}
	dna.selfProteins = dna.GenerateSelfProteins()
	switch dnaType {
	case HUMAN_DNA:
		dna.makeFunction = MakeStateDiagramByEukaryote
	case BACTERIA_DNA:
		dna.makeFunction = MakeStateDiagramByProkaryote
	}
	return dna
}

func (d *DNA) MHC_I() MHC_I {
	return &d.base.PublicKey
}

func HashProteins(proteins []Protein) [32]byte {
	b := make([]byte, 2*len(proteins))

	for i, protein := range proteins {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(protein))
	}

	return sha256.Sum256(b)
}

func (d *DNA) GenerateSelfProteins() map[Protein]bool {
	hash := d.GenerateAntigen([]Protein{42}).signature
	var proteins []Protein

	for i := 0; i < len(hash)/2; i++ {
		protein := Protein(binary.LittleEndian.Uint16(hash[i*2:]))
		proteins = append(proteins, protein)
	}

	selfProteins := make(map[Protein]bool)
	for _, protein := range proteins {
		selfProteins[protein] = true
	}
	return selfProteins
}

func (d *DNA) GenerateAntigen(proteins []Protein) *Antigen {
	hash := HashProteins(proteins)
	signature, err := ecdsa.SignASN1(rand.Reader, d.base, hash[:])
	if err != nil {
		log.Fatal(err)
	}
	return &Antigen{
		proteins:  proteins,
		signature: signature,
	}
}

func (d *DNA) Verify(a *ecdsa.PublicKey, m *Antigen) bool {
	hash := HashProteins(m.proteins)
	return ecdsa.VerifyASN1(a, hash[:], m.signature)
}
