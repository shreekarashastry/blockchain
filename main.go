package main

import (
	"github.com/shreekarashastry/blockchain/simulation"
)

const (
	numHonestMiners uint64 = 1
	numAdversary    uint64 = 1
)

func main() {
	simulation := simulation.NewSimulation(simulation.Bitcoin, numHonestMiners, numAdversary)
	simulation.Start()
}
