package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"log"
)

type DNAType elliptic.Curve
type MollecularPattern string

var HUMAN_DNA = DNAType(elliptic.P521())
var BACTERIA_DNA = DNAType(elliptic.P384())
var VIRUS_RNA = DNAType(elliptic.P224())

var BACTERIA_MOLECULAR_MOTIF = GetMollecularPattern(BACTERIA_DNA)
var VIRAL_MOLECULAR_MOTIF = GetMollecularPattern(VIRUS_RNA)

func GetMollecularPattern(dnaType DNAType) MollecularPattern {
	return MollecularPattern(dnaType.Params().P.String()[0:10])
}

var DNATypeMap = map[int]DNAType{
	521: HUMAN_DNA,
	384: BACTERIA_DNA,
	224: VIRUS_RNA,
}

type DNA struct {
	name         string
	base         *ecdsa.PrivateKey
	dnaType      DNAType
	selfProteins []Protein
	makeFunction func(c CellActor, dna *DNA) *StateDiagram
}
type MHC_I *ecdsa.PublicKey
type Protein uint16
type AntigenSignature []byte

type Antigen struct {
	proteins           []Protein
	signature          AntigenSignature
	mollecular_pattern MollecularPattern
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
	dna.Initialize()
	return dna
}

func MakeDNAFromRequest(request TransportRequest) (*DNA, error) {
	privateKey, err := x509.ParseECPrivateKey(request.Base)
	if err != nil {
		return nil, err
	}
	dNAType, ok := DNATypeMap[request.DNAType]
	if !ok {
		return nil, fmt.Errorf("cannot find DNA Type: %v", request.DNAType)
	}
	dna := &DNA{
		name:    request.Name,
		base:    privateKey,
		dnaType: dNAType,
	}
	dna.Initialize()
	return dna, nil
}

func (d *DNA) Initialize() {
	d.selfProteins = d.GenerateSelfProteins()
	switch d.dnaType {
	case HUMAN_DNA:
		d.makeFunction = MakeStateDiagramByEukaryote
	case BACTERIA_DNA:
		d.makeFunction = MakeStateDiagramByProkaryote
	}
}

func (d *DNA) Serialize() ([]byte, error) {
	return x509.MarshalECPrivateKey(d.base)
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

func (d *DNA) GenerateSelfProteins() []Protein {
	hash := d.GenerateAntigen([]Protein{42}).signature
	var proteins []Protein

	for i := 0; i < len(hash)/2; i++ {
		protein := Protein(binary.LittleEndian.Uint16(hash[i*2:]))
		proteins = append(proteins, protein)
	}
	return proteins
}

func (d *DNA) GenerateAntigen(proteins []Protein) *Antigen {
	hash := HashProteins(proteins)
	signature, err := ecdsa.SignASN1(rand.Reader, d.base, hash[:])
	if err != nil {
		log.Fatal(err)
	}
	return &Antigen{
		proteins:           proteins,
		signature:          signature,
		mollecular_pattern: GetMollecularPattern(d.dnaType),
	}
}

func (d *DNA) VerifySelf(a *ecdsa.PublicKey, m *Antigen) bool {
	hash := HashProteins(m.proteins)
	return ecdsa.VerifyASN1(a, hash[:], m.signature)
}
