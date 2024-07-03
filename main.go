package main

import (
	"github.com/shreekarashastry/blockchain/simulation"
)

const (
	numHonestMiners uint64 = 20
	numAdversary    uint64 = 5
)

func main() {
	blockchain := simulation.NewBlockchain()
	simulation := simulation.NewSimulation(blockchain, numHonestMiners, numAdversary)
	simulation.Start()
}
