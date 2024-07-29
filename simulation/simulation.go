package simulation

import (
	"fmt"
	"sync"
	"time"
)

const c_maxBlocks = 100

type Simulation struct {
	honestBc    *Blockchain
	adversaryBc *Blockchain

	honestStopCh    chan struct{}
	adversaryStopCh chan struct{}

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

func NewSimulation(bc, advBc *Blockchain, numHonestMiners, numAdversary uint64) *Simulation {
	engine := New()

	return &Simulation{
		honestBc:           bc,
		adversaryBc:        advBc,
		numHonestMiners:    numHonestMiners,
		numAdversary:       numAdversary,
		honestStopCh:       make(chan struct{}),
		adversaryStopCh:    make(chan struct{}),
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

	sim.wg.Add(4)
	go sim.honestMiningLoop()
	go sim.honestResultLoop()
	go sim.adversaryMiningLoop()
	go sim.adversaryResultLoop()

	// Both the honest miners and the aversary miners start mining, in this
	// simulation environment, I am approximating the independent miners to a go
	// routine
	firstPendingBlock := genesisBlock.PendingBlock()
	go func() {
		sim.honestNewWorkCh <- firstPendingBlock
	}()
	go func() {
		sim.adversaryNewWorkCh <- firstPendingBlock
	}()
	sim.wg.Wait()
}

func (sim *Simulation) Stop() {
	close(sim.quitCh)
}

func (sim *Simulation) interruptHonestWork() {
	if sim.honestStopCh != nil {
		close(sim.honestStopCh)
		sim.honestStopCh = nil
	}
}

func (sim *Simulation) interruptAdversaryWork() {
	if sim.adversaryStopCh != nil {
		close(sim.adversaryStopCh)
		sim.adversaryStopCh = nil
	}
}

func (sim *Simulation) honestMiningLoop() {
	defer sim.wg.Done()

	for {
		select {
		case newWork := <-sim.honestNewWorkCh:
			// If we reach the block number defined for the test
			if newWork.number <= c_maxBlocks {
				fmt.Println("Honest party finished the execution")
			}
			fmt.Println("New block to mine for honest party", newWork.number)
			for i := 0; i < int(sim.numHonestMiners); i++ {
				err := sim.engine.Seal(newWork, sim.honestMinedCh, sim.honestStopCh)
				if err != nil {
					fmt.Println("Error sealing the block", err)
				}
			}
		}
	}
}

func (sim *Simulation) honestResultLoop() {
	defer sim.wg.Done()
	for {
		select {
		case newBlock := <-sim.honestMinedCh:
			sim.interruptHonestWork()
			sim.honestStopCh = make(chan struct{})
			newBlock.SetTime(uint64(time.Now().Second()))
			sim.honestBc.blocks = append(sim.honestBc.blocks, newBlock)
			fmt.Println("Honest party Mined a new block", newBlock, "Hash", newBlock.Hash())
			sim.honestNewWorkCh <- newBlock.PendingBlock()
		case <-sim.quitCh:
			return
		}
	}
}

func (sim *Simulation) adversaryMiningLoop() {
	defer sim.wg.Done()

	for {
		select {
		case newWork := <-sim.adversaryNewWorkCh:
			// If we reach the block number defined for the test
			if newWork.number <= c_maxBlocks {
				fmt.Println("Adversary finished the execution")
			}
			fmt.Println("New block to mine adversary", newWork.number)
			for i := 0; i < int(sim.numAdversary); i++ {
				err := sim.engine.Seal(newWork, sim.adversaryMinedCh, sim.adversaryStopCh)
				if err != nil {
					fmt.Println("Error sealing the block", err)
				}
			}
		}
	}
}

func (sim *Simulation) adversaryResultLoop() {
	defer sim.wg.Done()
	for {
		select {
		case newBlock := <-sim.adversaryMinedCh:
			sim.interruptAdversaryWork()
			sim.adversaryStopCh = make(chan struct{})
			newBlock.SetTime(uint64(time.Now().Second()))
			sim.adversaryBc.blocks = append(sim.adversaryBc.blocks, newBlock)
			fmt.Println("Adversary Mined a new block", newBlock, "Hash", newBlock.Hash())
			// Add the timestamp details of when the block was mined and add the block to the blockchain
			sim.adversaryNewWorkCh <- newBlock.PendingBlock()
		case <-sim.quitCh:
			return
		}
	}
}
