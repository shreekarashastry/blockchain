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
	adversaryBlockchain := simulation.NewBlockchain()
	simulation := simulation.NewSimulation(blockchain, adversaryBlockchain, numHonestMiners, numAdversary)
	simulation.Start()
}
