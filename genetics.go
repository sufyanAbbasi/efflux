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
	mathRand "math/rand"
	"time"
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
	case VIRUS_RNA:
		d.makeFunction = MakeStateDiagramByVirus
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

	// Skip first two since they repeat.
	for i := 2; i < len(hash)/2; i++ {
		protein := Protein(binary.LittleEndian.Uint16(hash[i*2:]))
		proteins = append(proteins, protein)
	}
	return proteins
}

func (d *DNA) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = 42
	}
	return len(p), nil
}

func (d *DNA) GenerateAntigen(proteins []Protein) *Antigen {
	hash := HashProteins(proteins)
	// Generally, you would use rand.Reader as the first parameter to generate
	// a sufficiently salted signature scheme, but we want to make sure that the
	// protein signature generated every time is consistent so we use a dummy
	// reader that always returns 42.
	signature, err := ecdsa.SignASN1(d, d.base, hash[:])
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

func (d *DNA) GenerateNonselfProteins() (proteins []Protein) {
	selfProteinsMap := make(map[Protein]bool)
	for _, protein := range d.selfProteins {
		selfProteinsMap[protein] = true
	}
	for i := 0; i < 65535; i++ {
		_, isSelf := selfProteinsMap[Protein(i)]
		if !isSelf {
			proteins = append(proteins, Protein(i))
		}
	}
	return
}

func (d *DNA) Generate_MHCII_Groups(count int) (mhc_ii_groups []map[Protein]bool) {
	for i := 0; i < count; i++ {
		mhc_ii_groups = append(mhc_ii_groups, make(map[Protein]bool))
	}
	proteins := d.GenerateNonselfProteins()
	mathRand.Seed(time.Now().UnixNano())
	mathRand.Shuffle(len(proteins), func(i, j int) { proteins[i], proteins[j] = proteins[j], proteins[i] })
	for i, protein := range proteins {
		mhc_ii_groups[i%VIRGIN_TCELL_COUNT][protein] = true
	}
	return mhc_ii_groups
}
