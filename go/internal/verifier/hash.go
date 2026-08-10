package verifier

import "github.com/ethereum/go-ethereum/crypto"

func keccak256(data []byte) (h [32]byte) {
	copy(h[:], crypto.Keccak256(data))
	return h
}
