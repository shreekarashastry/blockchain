package simulation

import (
	"fmt"
	"sync"
	"time"
)

const (
	c_maxBlocks     = 15
	c_maxIterations = 50
)

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

	wg       sync.WaitGroup
	honestMu sync.RWMutex
	advMu    sync.RWMutex

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
		honestMinedCh:      make(chan *Block),
		honestNewWorkCh:    make(chan *Block),
		adversaryMinedCh:   make(chan *Block),
		adversaryNewWorkCh: make(chan *Block),
		engine:             engine,
		quitCh:             make(chan struct{}),
	}
}

func (sim *Simulation) Start() {
	winCounter := make([]int, c_maxBlocks)

	for i := 0; i < c_maxIterations; i++ {
		fmt.Println("Simulation Number", i)

		genesisBlock := GenesisBlock()

		sim.wg.Add(2)
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

		// Kill the test and create a new quitCh
		sim.Stop()
		sim.quitCh = make(chan struct{})

		// after this simulation is done, calculate a win chart
		for i := 1; i <= c_maxBlocks; i++ {
			if sim.honestBc.blocks[i].Time() < sim.adversaryBc.blocks[i].Time() {
				winCounter[i-1]++
			}
		}

		sim.honestBc = NewBlockchain()
		sim.adversaryBc = NewBlockchain()
	}

	fmt.Println("Win Counter", winCounter)
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
			if newWork.number > c_maxBlocks {
				fmt.Println("Honest party finished the execution")
				return
			}
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
	for {
		select {
		case newBlock := <-sim.honestMinedCh:
			sim.honestMu.Lock()
			sim.interruptHonestWork()
			sim.honestStopCh = make(chan struct{})
			newBlock.SetTime(uint64(time.Now().UnixMilli()))
			_, exists := sim.honestBc.blocks[int(newBlock.Number())]
			if !exists {
				sim.honestBc.blocks[int(newBlock.Number())] = newBlock
			}
			fmt.Println("Honest party Mined a new block", "Hash", newBlock.Number())
			sim.honestNewWorkCh <- newBlock.PendingBlock()
			sim.honestMu.Unlock()
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
			if newWork.number > c_maxBlocks {
				fmt.Println("Adversary finished the execution")
				return
			}
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
	for {
		select {
		case newBlock := <-sim.adversaryMinedCh:
			sim.advMu.Lock()
			sim.interruptAdversaryWork()
			sim.adversaryStopCh = make(chan struct{})
			newBlock.SetTime(uint64(time.Now().UnixMilli()))
			_, exists := sim.adversaryBc.blocks[int(newBlock.Number())]
			if !exists {
				sim.adversaryBc.blocks[int(newBlock.Number())] = newBlock
			}
			fmt.Println("Adversary Mined a new block", "Hash", newBlock.Number())
			// Add the timestamp details of when the block was mined and add the block to the blockchain
			sim.adversaryNewWorkCh <- newBlock.PendingBlock()
			sim.advMu.Unlock()
		case <-sim.quitCh:
			return
		}
	}
}
