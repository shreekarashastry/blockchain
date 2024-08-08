package simulation

import (
	"fmt"
	"sync"
	"time"

	"github.com/dominant-strategies/go-quai/event"
)

const (
	c_maxBlocks              = 5
	c_maxIterations          = 100
	c_honestDelta            = 10 // milliseconds
	c_commonPrefixFailure    = 0.1
	c_winningThreshold       = c_maxIterations * (1 - c_commonPrefixFailure)
	c_honestListeningThreads = 10
)

type Simulation struct {
	honestMiners []*Miner
	advMiners    []*Miner

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
		honestMiners = append(honestMiners, NewMiner(sim, &honestBlockFeed, HonestMiner, consensus))
	}
	sim.honestMiners = honestMiners

	for i := 0; i < int(numAdversary); i++ {
		advMiners = append(advMiners, NewMiner(sim, &advBlockFeed, AdversaryMiner, consensus))
	}
	sim.advMiners = advMiners
	return sim
}

func (sim *Simulation) Start() {
	// Start the honest miners
	for _, honestMiner := range sim.honestMiners {
		sim.wg.Add(1)
		honestMiner.Start()
	}
	for _, adversaryMiner := range sim.advMiners {
		sim.wg.Add(1)
		adversaryMiner.Start()
	}

	// Send the genesis block to mine
	sim.honestBlockFeed.Send(GenesisBlock())
	sim.advBlockFeed.Send(GenesisBlock())

	sim.wg.Wait()

	fmt.Println("Simulation done", sim.honestBc, sim.advBc)
}

func (sim *Simulation) Stop(minerType Kind) {
	if minerType == HonestMiner {
		for _, honestMiner := range sim.honestMiners {
			honestMiner.Stop()
			sim.wg.Done()
		}
	} else if minerType == AdversaryMiner {
		for _, adversaryMiner := range sim.advMiners {
			adversaryMiner.Stop()
			sim.wg.Done()
		}
	}
}
