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
	honestMiners []*Miner
	advMiners    []*Miner

	stopMu sync.RWMutex

	wg sync.WaitGroup

	honestBlockFeed *event.Feed
	advBlockFeed    *event.Feed

	simStartTime           time.Time
	simDuration            time.Duration
	totalHonestBlocks      uint64
	totalHonestSimDuration int64

	consensus Consensus // Bitcoin or Poem

	honestBc map[int]*Block
	advBc    map[int]*Block
}

func NewSimulation(consensus Consensus, numHonestMiners, numAdversary uint64) *Simulation {
	var honestBlockFeed event.Feed
	var advBlockFeed event.Feed
	// Create miiners and adversary
	honestMiners := make([]*Miner, 0)
	advMiners := make([]*Miner, 0)
	// Initialize the adversary miner
	sim := &Simulation{
		simStartTime:           time.Time{},
		simDuration:            0,
		totalHonestSimDuration: 0,
		totalHonestBlocks:      0,
		honestBlockFeed:        &honestBlockFeed,
		advBlockFeed:           &advBlockFeed,
		consensus:              consensus,
	}
	for i := 0; i < int(numHonestMiners); i++ {
		honestMiners = append(honestMiners, NewMiner(i, sim, &honestBlockFeed, HonestMiner, consensus))
	}
	sim.honestMiners = honestMiners

	for i := 0; i < int(numAdversary); i++ {
		advMiners = append(advMiners, NewMiner(i, sim, &advBlockFeed, AdversaryMiner, consensus))
	}
	sim.advMiners = advMiners
	return sim
}

func (sim *Simulation) Start() {
	winCounter := make([]int, c_maxBlocks)
	for i := 0; i < c_maxIterations; i++ {
		fmt.Println("Iteration", i)

		sim.simStartTime = time.Now()

		// Start the honest miners
		sim.wg.Add(1)
		for _, honestMiner := range sim.honestMiners {
			honestMiner.Start()
		}
		sim.wg.Add(1)
		for _, adversaryMiner := range sim.advMiners {
			adversaryMiner.Start()
		}

		// Send the genesis block to mine
		sim.honestBlockFeed.Send(GenesisBlock())
		sim.advBlockFeed.Send(GenesisBlock())

		sim.wg.Wait()
		// after this simulation is done, calculate a win chart
		for i := 1; i <= c_maxBlocks; i++ {
			honestBlock := sim.honestBc[i]
			adversaryBlock := sim.advBc[i]
			if honestBlock.Time() < adversaryBlock.Time() {
				winCounter[i-1]++
			}
		}
		sim.totalHonestSimDuration += sim.simDuration.Milliseconds()

		time.Sleep(3 * time.Second)
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
	fmt.Println("Simulation done")
}

func (sim *Simulation) Stop(minerType MinerKind) {
	sim.stopMu.Lock()
	defer sim.stopMu.Unlock()
	defer sim.wg.Done()
	if minerType == HonestMiner {
		if len(sim.honestBc) == c_maxBlocks-1 {
			return
		}
		for _, honestMiner := range sim.honestMiners {
			sim.simDuration = time.Since(sim.simStartTime)
			honestMiner.Stop()
		}
	} else if minerType == AdversaryMiner {
		if len(sim.advBc) == c_maxBlocks-1 {
			return
		}
		for _, adversaryMiner := range sim.advMiners {
			adversaryMiner.Stop()
		}
	}
}
