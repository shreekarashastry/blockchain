package main

import (
	"github.com/shreekarashastry/blockchain/simulation"
)

const (
	numHonestMiners uint64 = 20
	numAdversary    uint64 = 5
)

func main() {
	simulation := simulation.NewSimulation(simulation.Bitcoin, numHonestMiners, numAdversary)
	simulation.Start()
}
