package core

import (
	"crypto/elliptic"
	"encoding/gob"
	"io"

	"github.com/0maltsev/blockchain-from-scratch/crypto"
)

type Encoder[T any] interface {
	Encode(T) error
}

type Decoder[T any] interface {
	Decode(T) error
}

type GobTxEncoder struct {
	w io.Writer
}

func NewGobTxEncoder(w io.Writer) *GobTxEncoder {
	ellipticCurve := crypto.EncodingCurveInterface {
		Curve: elliptic.P256(),
	}
	gob.Register(ellipticCurve.Curve)
	return &GobTxEncoder{
		w: w,
	}
}

func (e *GobTxEncoder) Encode(tx *Transaction) error {
	return gob.NewEncoder(e.w).Encode(tx)
}

type GobTxDecoder struct {
	r io.Reader
}

func NewGobTxDecoder(r io.Reader) *GobTxDecoder {
	ellipticCurve := crypto.EncodingCurveInterface {
		Curve: elliptic.P256(),
	}
	gob.Register(ellipticCurve.Curve)
	return &GobTxDecoder{
		r: r,
	}
}

func (e *GobTxDecoder) Decode(tx *Transaction) error {
	return gob.NewDecoder(e.r).Decode(tx)
}
