package simulation

import (
	crand "crypto/rand"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
)

var (
	big2e256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0)) // 2^256
)

type Blake3pow struct {
	rand    *rand.Rand // For random seeding of the source for nonce
	threads int
	lock    sync.Mutex
}

func New() *Blake3pow {
	blake3pow := &Blake3pow{
		threads: 1,
	}
	return blake3pow
}

// Seal implements consensus.Engine, attempting to find a nonce that satisfies
// the header's difficulty requirements.
func (blake3pow *Blake3pow) Seal(header *Block, results chan<- *Block, stop <-chan struct{}) error {
	// Create a runner and the multiple search threads it directs
	abort := make(chan struct{})

	blake3pow.lock.Lock()
	threads := blake3pow.threads
	if blake3pow.rand == nil {
		seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			blake3pow.lock.Unlock()
			return err
		}
		blake3pow.rand = rand.New(rand.NewSource(seed.Int64()))
	}
	blake3pow.lock.Unlock()
	if threads == 0 {
		threads = runtime.NumCPU()
	}
	if threads < 0 {
		threads = 0 // Allows disabling local mining without extra logic around local/remote
	}
	var (
		pend   sync.WaitGroup
		locals = make(chan *Block)
	)
	for i := 0; i < threads; i++ {
		pend.Add(1)
		go func(id int, nonce uint64) {
			defer pend.Done()
			blake3pow.mine(header, id, nonce, abort, locals)
		}(i, uint64(blake3pow.rand.Int63()))
	}
	// Wait until sealing is terminated or a nonce is found
	go func() {
		var result *Block
		select {
		case <-stop:
			// Outside abort, stop all miner threads
			close(abort)
		case result = <-locals:
			// One of the threads found a block, abort all others
			select {
			case results <- result:
			default:
				fmt.Println("Sealing result is not read by miner")
			}
			close(abort)
		}
		// Wait for all miners to terminate and return the block
		pend.Wait()
	}()
	return nil
}

// mine is the actual proof-of-work miner that searches for a nonce starting from
// seed that results in correct final header difficulty.
func (blake3pow *Blake3pow) mine(header *Block, id int, seed uint64, abort chan struct{}, found chan *Block) {
	// Extract some data from the header
	var (
		target = new(big.Int).Div(big2e256, big.NewInt(int64(header.Difficulty())))
	)

	// Start generating random nonces until we abort or find a good one
	var (
		attempts  = int64(0)
		nonce     = seed
		powBuffer = new(big.Int)
	)
search:
	for {
		select {
		case <-abort:
			// Mining terminated, update stats and abort
			break search

		default:
			// We don't have to update hash rate on every nonce, so update after after 2^X nonces
			attempts++
			if (attempts % (1 << 15)) == 0 {
				attempts = 0
			}
			// Compute the PoW value of this nonce
			header.SetNonce(EncodeNonce(nonce))
			hash := header.Hash().Bytes()
			if powBuffer.SetBytes(hash).Cmp(target) <= 0 {
				// Correct nonce found, create a new header with it
				// Seal and return a block (if still needed)
				select {
				case found <- header:
				case <-abort:
				}
				break search
			}
			nonce++
		}
	}
}
