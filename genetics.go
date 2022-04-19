package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	mRand "math/rand"
)

var HUMAN_DNA = elliptic.P521()
var BACTERIA_DNA = elliptic.P384()
var VIRUS_RNA = elliptic.P224()

type DNAType elliptic.Curve
type DNA struct {
	name             string
	base             *ecdsa.PrivateKey
	dnaType          DNAType
	antigenSignature AntigenSignature
}
type MHC_I *ecdsa.PublicKey
type AntigenSignature uint16
type Antigen []byte

func makeDNA(dnaType DNAType, name string) *DNA {
	// Caution: slow!
	privateKey, err := ecdsa.GenerateKey(dnaType, rand.Reader)
	if err != nil {
		panic(err)
	}
	return &DNA{
		name:             name,
		base:             privateKey,
		dnaType:          dnaType,
		antigenSignature: AntigenSignature(mRand.Uint32()),
	}
}

func (d *DNA) MHC_I() MHC_I {
	return &d.base.PublicKey
}

func (d *DNA) GenerateAntigen() Antigen {
	hash := sha256.Sum256([]byte{byte(d.antigenSignature)})
	sig, err := ecdsa.SignASN1(rand.Reader, d.base, hash[:])
	if err != nil {
		panic(err)
	}
	return sig
}

func (d *DNA) Verify(a *ecdsa.PublicKey, m Antigen) bool {
	hash := sha256.Sum256([]byte{byte(d.antigenSignature)})
	return ecdsa.VerifyASN1(a, hash[:], m)
}
