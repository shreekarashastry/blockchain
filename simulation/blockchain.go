package simulation

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"fmt"

	"lukechampine.com/blake3"
)

const HashLength = 32

type Hash [HashLength]byte

type BlockNonce [8]byte

// EncodeNonce converts the given integer to a block nonce.
func EncodeNonce(i uint64) BlockNonce {
	var n BlockNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// Bytes() returns the raw bytes of the block nonce
func (n BlockNonce) Bytes() []byte {
	return n[:]
}

// Uint64 returns the integer value of a block nonce.
func (n BlockNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

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
	parentHash Hash
	number     uint64
	difficulty uint64
	nonce      [8]byte
	time       uint64
}

func GenesisBlock() *Block {
	return &Block{
		number:     0,
		difficulty: 1000000,
	}
}

func (b *Block) PendingBlock() *Block {
	return &Block{
		parentHash: b.Hash(),
		number:     b.Number() + 1,
		difficulty: b.Difficulty(),
	}
}

func (b *Block) Hash() (hash Hash) {
	sealHash := b.SealHash().Bytes()
	var hData [40]byte
	copy(hData[:], b.Nonce().Bytes())
	copy(hData[len(b.nonce):], sealHash)
	sum := blake3.Sum256(hData[:])
	hash.SetBytes(sum[:])
	return hash
}

func (b *Block) SealHash() (hash Hash) {
	sealData := struct {
		ParentHash Hash
		Number     uint64
		Difficulty uint64
	}{
		ParentHash: b.ParentHash(),
		Number:     b.Number(),
		Difficulty: b.Difficulty(),
	}
	buf := bytes.Buffer{}
	e := gob.NewEncoder(&buf)
	err := e.Encode(sealData)
	if err != nil {
		fmt.Println(`failed gob Encode`, err)
	}
	data := buf.Bytes()
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

func (b *Block) Nonce() BlockNonce {
	return b.nonce
}

func (b *Block) Time() uint64 {
	return b.time
}
func (b *Block) SetNonce(nonce BlockNonce) {
	b.nonce = nonce
}

func (b *Block) String() string {
	return fmt.Sprintf("{ ParentHash: %v, Number: %v, Difficulty %v, Nonce: %v, Time: %v}", b.ParentHash(), b.Number(), b.Difficulty(), b.Nonce(), b.Time())
}

func NewBlockchain() *Blockchain {
	return &Blockchain{
		blocks: make([]*Block, 0),
	}
}
