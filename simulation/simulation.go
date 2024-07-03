package simulation

import (
	"fmt"
	"sync"
)

type Simulation struct {
	blockchain      *Blockchain
	numHonestMiners uint64
	numAdversary    uint64

	honestMinedCh   chan *Block
	honestNewWorkCh chan *Block

	adversaryMinedCh   chan *Block
	adversaryNewWorkCh chan *Block

	wg sync.WaitGroup

	quitCh chan struct{}

	engine *Blake3pow
}

func NewSimulation(bc *Blockchain, numHonestMiners, numAdversary uint64) *Simulation {
	engine := New()

	return &Simulation{
		blockchain:         bc,
		numHonestMiners:    numHonestMiners,
		numAdversary:       numAdversary,
		honestMinedCh:      make(chan *Block, 10),
		honestNewWorkCh:    make(chan *Block, 10),
		adversaryMinedCh:   make(chan *Block, 10),
		adversaryNewWorkCh: make(chan *Block, 10),
		engine:             engine,
	}
}

func (sim *Simulation) Start() {
	genesisBlock := GenesisBlock()

	fmt.Println("Genesis Block", genesisBlock.Hash().String())

	sim.wg.Add(2)
	go sim.miningLoop()
	go sim.resultLoop()

	firstPendingBlock := genesisBlock.PendingBlock()
	sim.honestNewWorkCh <- firstPendingBlock

	sim.wg.Wait()
}

func (sim *Simulation) Stop() {
	close(sim.quitCh)
}

func (sim *Simulation) miningLoop() {
	defer sim.wg.Done()

	for {
		select {
		case newWork := <-sim.honestNewWorkCh:
			fmt.Println("New block to mine", newWork)
			err := sim.engine.Seal(newWork, sim.honestMinedCh, sim.quitCh)
			if err != nil {
				fmt.Println("Error sealing the block", err)
			}
		}
	}
}

func (sim *Simulation) resultLoop() {
	defer sim.wg.Done()
	for {
		select {
		case newBlock := <-sim.honestMinedCh:
			fmt.Println("New Block Mined", newBlock, "Hash", newBlock.Hash())
		case <-sim.quitCh:
			return
		}
	}
}
