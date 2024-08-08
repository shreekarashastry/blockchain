package simulation

import (
	"fmt"
	"sync"
	"time"

	"github.com/dominant-strategies/go-quai/event"
)

type Kind uint

const (
	HonestMiner Kind = iota
	AdversaryMiner
)

type Miner struct {
	bc            *BlockDB
	currentHead   *Block
	engine        *Blake3pow
	minedCh       chan *Block
	newBlockCh    chan *Block
	newBlockSub   event.Subscription
	stopCh        chan struct{}
	broadcastFeed *event.Feed
	minerType     Kind
	consensus     Consensus
	sim           *Simulation
	lock          sync.RWMutex
}

func NewMiner(sim *Simulation, broadcastFeed *event.Feed, kind Kind, consensus Consensus) *Miner {
	return &Miner{
		bc:            NewBlockchain(),
		engine:        New(),
		minedCh:       make(chan *Block, 10),
		stopCh:        make(chan struct{}),
		broadcastFeed: broadcastFeed,
		newBlockCh:    make(chan *Block, 10),
		minerType:     kind,
		consensus:     consensus,
		currentHead:   GenesisBlock(),
		sim:           sim,
	}
}

func (m *Miner) Start() {
	m.newBlockSub = m.SubscribeMinedBlocksEvent()
	go m.ListenNewBlocks()
	go m.MinedEvent()
}

func (m *Miner) Stop() {
	m.newBlockSub.Unsubscribe()
}

func (m *Miner) interruptMining() {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.stopCh != nil {
		close(m.stopCh)
		m.stopCh = nil
	}
}

func (m *Miner) ListenNewBlocks() {
	for {
		select {
		case newBlock := <-m.newBlockCh:
			// If this block already is in the database, we mined it
			_, exists := m.bc.blocks.Get(newBlock.Hash())
			if !exists {
				// Once a new block is mined, the current miner starts mining the
				// new block
				m.interruptMining()
				m.SetCurrentHead(newBlock)
				m.Mine()
			}
		case <-m.newBlockSub.Err():
			return
		}
	}
}

func (m *Miner) MinedEvent() {
	for {
		select {
		case minedBlock := <-m.minedCh:
			fmt.Println("Mined a new block", m.minerType, minedBlock.Hash(), minedBlock.Number())
			// Add block to the block database
			m.bc.blocks.Add(minedBlock.Hash(), *CopyBlock(minedBlock))

			// Once a new block is mined, the current miner starts mining the
			// new block
			m.interruptMining()
			m.SetCurrentHead(minedBlock)

			// If we hit the max block, stop all miners of the same kind to stop
			if minedBlock.Number() >= c_maxBlocks {
				switch m.minerType {
				case HonestMiner:
					m.sim.honestBc = m.ConstructBlockchain()
				case AdversaryMiner:
					m.sim.advBc = m.ConstructBlockchain()
				}
				m.sim.Stop(m.minerType)
			}

			// Start mining the next block
			m.Mine()

			// In the case of the honest miner add a broadcast delay
			if m.minerType == HonestMiner {
				time.Sleep(c_honestDelta * time.Millisecond)
			}

			m.broadcastFeed.Send(minedBlock)

		case <-m.newBlockSub.Err():
			return
		}
	}
}

func (m *Miner) ConstructBlockchain() map[int]*Block {
	bc := make(map[int]*Block)
	currentHead := m.currentHead
	bc[int(currentHead.Number())] = currentHead
	for {
		if currentHead.ParentHash() == GenesisBlock().Hash() {
			break
		}
		parent, exists := m.bc.blocks.Get(currentHead.ParentHash())
		if !exists {
			panic("parent doesnt exist")
		}
		currentHead = CopyBlock(&parent)
		bc[int(parent.Number())] = CopyBlock(&parent)
	}
	return bc
}

func (m *Miner) CalculateBlockWeight(block *Block) float64 {
	if m.consensus == Bitcoin {
		return block.ParentWeight() + float64(block.Difficulty())
	} else if m.consensus == Poem {
		return block.ParentWeight() + m.engine.IntrinsicDifficulty(block)
	} else {
		panic("invalid consensus type")
	}
}

func (m *Miner) SetCurrentHead(block *Block) {
	// If we are trying to the current head again return
	if m.currentHead.Hash() == block.Hash() {
		return
	}

	// We set the first block
	if block.ParentHash() == GenesisBlock().Hash() {
		m.currentHead = CopyBlock(block)
		return
	}

	currentHeadWeight := m.CalculateBlockWeight(m.currentHead)
	newBlockWeight := m.CalculateBlockWeight(block)

	if newBlockWeight > currentHeadWeight {
		m.currentHead = CopyBlock(block)
	}
}

func (m *Miner) Mine() {
	newPendingHeader := m.currentHead.PendingBlock()
	newPendingHeader.SetParentWeight(m.CalculateBlockWeight(m.currentHead))

	m.stopCh = make(chan struct{})
	err := m.engine.Seal(newPendingHeader, m.minedCh, m.stopCh)
	if err != nil {
		fmt.Println("Error sealing the block", err)
	}
}

func (m *Miner) SubscribeMinedBlocksEvent() event.Subscription {
	return m.broadcastFeed.Subscribe(m.newBlockCh)
}
