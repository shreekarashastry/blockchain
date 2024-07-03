package main

import (
	"encoding/json"
	"sync/atomic"

	"lukechampine.com/blake3"
)

const HashLength = 20

type Hash [20]byte

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

func (h Hash) Bytes() []byte {
	return h[:]
}

type Blockchain struct {
	blocks []*Block
}

type Block struct {
	parentHash Hash   `json:"parentHash"`
	number     uint64 `json:"number"`
	difficulty uint64 `json:"difficulty"`
	nonce      uint64 `json:"nonce"`
	time       atomic.Value
}

func GenesisBlock() *Block {
	return &Block{
		number:     0,
		difficulty: 100000,
		nonce:      100,
	}
}

func (b *Block) Hash() (hash Hash) {
	data, _ := json.Marshal(b)
	sum := blake3.Sum256(data[:])
	hash.SetBytes(sum[:])
	return hash
}

func (b *Block) ParentHash() Hash {
	return b.parentHash
}

func (b *Block) Number() uint64 {
	return b.number
}

func (b *Block) Difficulty() uint64 {
	return b.difficulty
}

func (b *Block) SetNonce(nonce uint64) {
	b.nonce = nonce
}

func NewBlockchain() *Blockchain {
	return &Blockchain{
		blocks: make([]*Block, 0),
	}
}
