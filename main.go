package main

const (
	numHonestMiners uint64 = 20
	numAdversary    uint64 = 5
)

func main() {
	blockchain := NewBlockchain()
	simulation := NewSimulation(blockchain, numHonestMiners, numAdversary)
	simulation.Start()
}
