package simulation

import (
	"fmt"
	"sync"
	"time"

	"github.com/dominant-strategies/go-quai/event"
)

const (
	c_maxBlocks              = 100
	c_maxIterations          = 100
	c_honestDelta            = 140 // milliseconds
	c_commonPrefixFailure    = 0.1
	c_winningThreshold       = c_maxIterations * (1 - c_commonPrefixFailure)
	c_honestListeningThreads = 10
)

type Simulation struct {
	honestMiners    []*Miner
	advMiners       []*Miner
	honestStopCh    chan struct{}
	adversaryStopCh chan struct{}

	honestMinedCh   chan *Block
	honestNewWorkCh chan *Block

	adversaryMinedCh   chan *Block
	adversaryNewWorkCh chan *Block

	miningWg sync.WaitGroup

	honestMu sync.RWMutex

	quitCh chan struct{}

	scope        event.SubscriptionScope
	newBlockFeed event.Feed

	simStartTime           time.Time
	simDuration            time.Duration
	totalHonestBlocks      uint64
	totalHonestSimDuration int64
}

func NewSimulation(bc, advBc *Blockchain, numHonestMiners, numAdversary uint64) *Simulation {
	// Create miiners and adversary
	honestMinedCh := make(chan *Block, 20)
	honestMiners := make([]*Miner, 0)
	advMiners := make([]*Miner, 0)
	for i := 0; i < int(numHonestMiners); i++ {
		honestMiners = append(honestMiners, NewMiner(honestMinedCh))
	}
	for i := 0; i < int(numAdversary); i++ {
		advMiners = append(advMiners, NewMiner())
	}

	return &Simulation{
		honestMiners:           honestMiners,
		advMiners:              advMiners,
		honestNewWorkCh:        make(chan *Block),
		adversaryMinedCh:       make(chan *Block, 20),
		adversaryNewWorkCh:     make(chan *Block),
		quitCh:                 make(chan struct{}),
		simStartTime:           time.Time{},
		simDuration:            0,
		totalHonestSimDuration: 0,
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
		go sim.honestResultLoop()
		go sim.adversaryResultLoop()

		// Both the honest miners and the aversary miners start mining, in this
		// simulation environment, I am approximating the independent miners to a go
		// routine
		firstPendingBlock := genesisBlock.PendingBlock()
		go func() { sim.honestNewWorkCh <- firstPendingBlock }()
		go func() { sim.adversaryNewWorkCh <- firstPendingBlock }()
		sim.miningWg.Wait()

		// Kill the test
		sim.Stop()

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
	for {
		select {
		case newWork := <-sim.honestNewWorkCh:
			sim.honestStopCh = make(chan struct{})
			for i := 0; i < int(sim.numHonestMiners); i++ {
			}
		case <-sim.quitCh:
			return
		}
	}
}

func (sim *Simulation) honestResultLoop() {
	defer sim.miningWg.Done()

	var wg sync.WaitGroup
	honestWorkerStopChan := make(chan struct{})

	worker := func(quit <-chan struct{}) {
		defer wg.Done()
		for {
			select {
			case honestBlock := <-sim.honestMinedCh:
				fmt.Println("Honest party Mined a new block", "Hash", honestBlock.Number())
				sim.totalHonestBlocks++
				// simulating network delay
				time.Sleep(c_honestDelta * time.Millisecond)
				_, exists := sim.honestBc.blocks.Get(honestBlock.Number())
				if !exists {
					// sleep for time defined for this experiment
					sim.interruptHonestWork()
					honestBlock.SetTime(uint64(time.Now().UnixMilli()))
					sim.honestBc.blocks.Add(honestBlock.Number(), *honestBlock)
					// Dont mine more blocks after reaching the max blocks
					if honestBlock.Number() < c_maxBlocks {
						select {
						case sim.honestNewWorkCh <- honestBlock.PendingBlock():
						default:
						}
					} else {
						sim.simDuration = time.Since(sim.simStartTime)
						// exit all workers
						close(honestWorkerStopChan)
					}
				}
			case <-quit:
				return
			}
		}
	}

	for i := 0; i < c_honestListeningThreads; i++ {
		wg.Add(1)
		go worker(honestWorkerStopChan)
	}

	wg.Wait()
}

func (sim *Simulation) adversaryMiningLoop() {
	for {
		select {
		case newWork := <-sim.adversaryNewWorkCh:
			sim.adversaryStopCh = make(chan struct{})
			for i := 0; i < int(sim.numAdversary); i++ {
				err := sim.engine.Seal(newWork, sim.adversaryMinedCh, sim.adversaryStopCh)
				if err != nil {
					fmt.Println("Error sealing the block", err)
				}
			}

		case <-sim.quitCh:
			return
		}
	}
}

func (sim *Simulation) adversaryResultLoop() {
	defer sim.miningWg.Done()
	for {
		select {
		case newBlock := <-sim.adversaryMinedCh:
			fmt.Println("Adversary Mined a new block", "Hash", newBlock.Number())
			_, exists := sim.adversaryBc.blocks.Get(newBlock.Number())
			if !exists {
				sim.interruptAdversaryWork()
				newBlock.SetTime(uint64(time.Now().UnixMilli()))
				sim.adversaryBc.blocks.Add(newBlock.Number(), *newBlock)
				// Dont mine more blocks after reaching the max blocks
				if newBlock.Number() < c_maxBlocks {
					select {
					case sim.adversaryNewWorkCh <- newBlock.PendingBlock():
					default:
					}
				} else {
					return
				}
			}
		}
	}
}
