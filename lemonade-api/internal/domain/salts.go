package domain

import "math/rand"

// Salts give each random system its own stream: rand(seed ^ day ^ salt). One salt
// per system means adding a system never changes what an existing system draws,
// so market prices for an existing seed stay the same when new content lands.
// Every new system that rolls dice declares its salt here and adds a test that the
// market walk for a fixed seed is unchanged.
const (
	// SaltMarket is the original stream, shared by the event roll and then the
	// price walk, in that order. It is 0 so games started before salts existed
	// replay exactly.
	SaltMarket int64 = 0
)

// dayRNG returns the random stream for one system on one day.
func dayRNG(seed int64, day int, salt int64) *rand.Rand {
	return rand.New(rand.NewSource(seed ^ int64(day) ^ salt))
}
