package domain

import "math/rand"

// walk applies one day of the mean-reverting random walk, clamped to
// [ClampMin, ClampMax] x base (SPEC rule 18).
func walk(rng *rand.Rand, price float64, base int, cfg Config) float64 {
	next := price + cfg.RevertRate*(float64(base)-price) + price*cfg.Sigma*rng.NormFloat64()

	min := cfg.ClampMin * float64(base)
	max := cfg.ClampMax * float64(base)
	if next < min {
		next = min
	}
	if next > max {
		next = max
	}
	return next
}
