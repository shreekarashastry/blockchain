package simulation

import (
	"fmt"
)

type Miner struct {
	bc      *Blockchain
	engine  *Blake3pow
	minedCh chan *Block
	stopCh  chan struct{}
}

func NewMiner(minedCh chan *Block) *Miner {
	return &Miner{
		bc:      NewBlockchain(),
		engine:  New(),
		minedCh: minedCh,
		stopCh:  make(chan struct{}),
	}
}

func (m *Miner) ListenNewBlocks(newBlockCh chan *Block) {
	for {
		select {
		case newPendingHeader := <-newBlockCh:
			m.Mine(newPendingHeader)
		}
	}
}

func (m *Miner) Mine(newPendingHeader *Block) {
	err := m.engine.Seal(newPendingHeader, m.minedCh, m.stopCh)
	if err != nil {
		fmt.Println("Error sealing the block", err)
	}
}

func (m *Miner) SubscribeMinedBlocksEvent() {}
