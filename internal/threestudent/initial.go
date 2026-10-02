package threestudent

import (
	"encoding/binary"
	"math"
	"math/rand"
)

const InitialSHA = "fa8af3f51b719cd11b99cd4ff7f89f64582695dc89ead5c2e09701c187a6267f"

// InitialWeights is new seeded uniform initialization, independent of every
// teacher/pretrained/student file. Layout is w1,b1,w2,b2, little-endian FP32.
func InitialWeights() []byte {
	rng := rand.New(rand.NewSource(InitialSeed))
	result := make([]byte, 74624)
	at := 0
	for _, tensor := range [4][2]int{{768 * 24, 768}, {24, 768}, {24 * 8, 24}, {8, 24}} {
		bound := 1 / math.Sqrt(float64(tensor[1]))
		for range tensor[0] {
			v := float32((2*rng.Float64() - 1) * bound)
			binary.LittleEndian.PutUint32(result[at:], math.Float32bits(v))
			at += 4
		}
	}
	return result
}
