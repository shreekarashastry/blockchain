package simulation

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"

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

func (h Hash) String() string {
	enc := make([]byte, len(h[:])*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], h[:])
	return string(enc)
}

func (h Hash) Bytes() []byte {
	return h[:]
}

type Blockchain struct {
	blocks []*Block
}

type Block struct {
	PHash     Hash
	Num       uint64
	Diff      uint64
	Nnce      uint64
	BlockTime uint64
}

func GenesisBlock() *Block {
	return &Block{
		Num:  0,
		Diff: 100000000000,
		Nnce: 100,
	}
}

func (b *Block) PendingBlock() *Block {
	return &Block{
		PHash: b.Hash(),
		Num:   b.Num + 1,
		Diff:  b.Diff,
	}
}

func (b *Block) Hash() (hash Hash) {
	buf := bytes.Buffer{}
	e := gob.NewEncoder(&buf)
	err := e.Encode(b)
	if err != nil {
		fmt.Println(`failed gob Encode`, err)
	}
	data := buf.Bytes()
	sum := blake3.Sum256(data[:])
	hash.SetBytes(sum[:])
	return hash
}

func (b *Block) ParentHash() Hash {
	return b.PHash
}

func (b *Block) Number() uint64 {
	return b.Num
}

func (b *Block) Difficulty() uint64 {
	return b.Diff
}

func (b *Block) Nonce() uint64 {
	return b.Nnce
}

func (b *Block) Time() uint64 {
	return b.BlockTime
}
func (b *Block) SetNonce(nonce uint64) {
	b.Nnce = nonce
}

func (b *Block) String() string {
	return fmt.Sprintf("{ ParentHash: %v, Number: %v, Difficulty %v, Nonce: %v, Time: %v}", b.ParentHash(), b.Number(), b.Difficulty(), b.Nonce(), b.Time())
}

func NewBlockchain() *Blockchain {
	return &Blockchain{
		blocks: make([]*Block, 0),
	}
}
