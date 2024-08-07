package simulation

import (
	"fmt"
	"sync"
	"time"
)

const (
	c_maxBlocks              = 100
	c_maxIterations          = 100
	c_honestDelta            = 120 // milliseconds
	c_commonPrefixFailure    = 0.1
	c_winningThreshold       = c_maxIterations * (1 - c_commonPrefixFailure)
	c_honestListeningThreads = 10
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

	miningWg  sync.WaitGroup
	newWorkWg sync.WaitGroup

	honestMu sync.RWMutex

	quitCh chan struct{}

	engine *Blake3pow

	simStartTime           time.Time
	simDuration            time.Duration
	numHonestBlocks        uint64
	totalHonestBlocks      uint64
	totalHonestSimDuration int64
}

func NewSimulation(bc, advBc *Blockchain, numHonestMiners, numAdversary uint64) *Simulation {
	engine := New()

	return &Simulation{
		honestBc:               bc,
		adversaryBc:            advBc,
		numHonestMiners:        numHonestMiners,
		numAdversary:           numAdversary,
		honestStopCh:           make(chan struct{}),
		adversaryStopCh:        make(chan struct{}),
		honestMinedCh:          make(chan *Block, 20),
		honestNewWorkCh:        make(chan *Block),
		adversaryMinedCh:       make(chan *Block, 20),
		adversaryNewWorkCh:     make(chan *Block),
		engine:                 engine,
		quitCh:                 make(chan struct{}),
		simStartTime:           time.Time{},
		simDuration:            0,
		totalHonestSimDuration: 0,
		numHonestBlocks:        0,
		totalHonestBlocks:      0,
	}
}

func (sim *Simulation) Start() {
	winCounter := make([]int, c_maxBlocks)

	for i := 0; i < c_maxIterations; i++ {
		fmt.Println("Simulation Number", i)

		genesisBlock := GenesisBlock()

		sim.simStartTime = time.Now()

		sim.miningWg.Add(2)
		go sim.honestMiningLoop()
		go sim.adversaryMiningLoop()

		sim.newWorkWg.Add(2)
		go sim.honestResultLoop()
		go sim.adversaryResultLoop()

		// Both the honest miners and the aversary miners start mining, in this
		// simulation environment, I am approximating the independent miners to a go
		// routine
		firstPendingBlock := genesisBlock.PendingBlock()
		go func() { sim.honestNewWorkCh <- firstPendingBlock }()
		go func() { sim.adversaryNewWorkCh <- firstPendingBlock }()
		sim.miningWg.Wait()

		// Kill the test and create a new quitCh
		sim.Stop()
		sim.newWorkWg.Wait()

		// after this simulation is done, calculate a win chart
		for i := 1; i <= c_maxBlocks; i++ {
			honestBlock, _ := sim.honestBc.blocks.Get(uint64(i))
			adversaryBlock, _ := sim.adversaryBc.blocks.Get(uint64(i))
			if honestBlock.Time() < adversaryBlock.Time() {
				winCounter[i-1]++
			}
		}

		sim.honestBc = NewBlockchain()
		sim.adversaryBc = NewBlockchain()
		sim.simStartTime = time.Time{}
		sim.totalHonestSimDuration += sim.simDuration.Milliseconds()
		fmt.Println("Sim duration", sim.simDuration)
		sim.simDuration = 0
		sim.quitCh = make(chan struct{})
		sim.numHonestBlocks = 0
	}

	avgHonestBlocks := sim.totalHonestBlocks / c_maxIterations
	avgHonestRoundTime := sim.totalHonestSimDuration / (c_maxIterations * c_honestDelta)
	fmt.Println("Simulation Summary")
	fmt.Println("Honest Time Delta", c_honestDelta, "milliseconds")
	fmt.Println("Average num of honest blocks", avgHonestBlocks)
	fmt.Println("Average honest sim duration in Delta", avgHonestRoundTime)

	g := float64(avgHonestBlocks) / float64(avgHonestRoundTime)
	f := float64(c_maxBlocks) / float64(avgHonestRoundTime)
	var k uint64
	for i := 0; i < len(winCounter); i++ {
		if winCounter[i] > c_winningThreshold {
			k = uint64(i) + 1
			break
		}
	}
	d := float64(k) / f

	fmt.Println("win counter", winCounter)
	fmt.Println("g", g, "f", f, "k", k, "d", d)
}

func (sim *Simulation) Stop() {
	if sim.quitCh != nil {
		close(sim.quitCh)
	}
}

func (sim *Simulation) interruptHonestWork() {
	sim.honestMu.Lock()
	defer sim.honestMu.Unlock()
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
	defer sim.miningWg.Done()

	for {
		select {
		case newWork := <-sim.honestNewWorkCh:
			sim.numHonestBlocks++
			sim.totalHonestBlocks++
			// If we reach the block number defined for the test
			if newWork.number > c_maxBlocks {
				fmt.Println("Honest party finished the execution")
				sim.simDuration = time.Since(sim.simStartTime)
				return
			}
			sim.honestStopCh = make(chan struct{})
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
	defer sim.newWorkWg.Done()
	quit := make(chan struct{})
	var wg sync.WaitGroup

	worker := func(quit <-chan struct{}) {
		defer wg.Done()
		for {
			select {
			case honestBlock := <-sim.honestMinedCh:
				// simulating network delay
				time.Sleep(c_honestDelta * time.Millisecond)
				_, exists := sim.honestBc.blocks.Get(honestBlock.Number())
				if !exists {
					// sleep for time defined for this experiment
					sim.interruptHonestWork()
					honestBlock.SetTime(uint64(time.Now().UnixMilli()))
					sim.honestBc.blocks.Add(honestBlock.Number(), *honestBlock)
					select {
					case sim.honestNewWorkCh <- honestBlock.PendingBlock():
					default:
					}
				} else {
					if honestBlock.Number() > c_maxBlocks {
						return
					}
				}
				fmt.Println("Honest party Mined a new block", "Hash", honestBlock.Number())
			case <-quit:
				return
			}
		}
	}

	for i := 0; i < c_honestListeningThreads; i++ {
		wg.Add(1)
		go worker(quit)
	}

	<-sim.quitCh
	close(quit)
	wg.Wait()
}

func (sim *Simulation) adversaryMiningLoop() {
	defer sim.miningWg.Done()

	for {
		select {
		case newWork := <-sim.adversaryNewWorkCh:
			if newWork == nil {
				return
			}
			// If we reach the block number defined for the test
			if newWork.number > c_maxBlocks {
				fmt.Println("Adversary finished the execution")
				return
			}
			sim.adversaryStopCh = make(chan struct{})
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
	defer sim.newWorkWg.Done()
	for {
		select {
		case newBlock := <-sim.adversaryMinedCh:
			sim.interruptAdversaryWork()
			_, exists := sim.adversaryBc.blocks.Get(newBlock.Number())
			if !exists {
				newBlock.SetTime(uint64(time.Now().UnixMilli()))
				sim.adversaryBc.blocks.Add(newBlock.Number(), *newBlock)
				select {
				case sim.adversaryNewWorkCh <- newBlock.PendingBlock():
				default:
				}
			}
			fmt.Println("Adversary Mined a new block", "Hash", newBlock.Number())
		case <-sim.quitCh:
			return
		}
	}
}
