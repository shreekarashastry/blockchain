package main

import "fmt"

type Simulation struct {
	blockchain      *Blockchain
	numHonestMiners uint64
	numAdversary    uint64
}

func NewSimulation(bc *Blockchain, numHonestMiners, numAdversary uint64) *Simulation {
	return &Simulation{blockchain: bc, numHonestMiners: numHonestMiners, numAdversary: numAdversary}
}

func (sim *Simulation) Start() {
	genesisBlock := GenesisBlock()

	fmt.Println("Genesis Block", genesisBlock.Hash())
}
