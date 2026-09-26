package domain

import "lemonade-api/internal/domain/content"

// The tests were written against the fixed list of five commodities. They read the
// same lists from the default catalog, so they keep testing the real game.
var (
	// The original five: bots and the golden runs play the base game, and the launch
	// commodities are tested on their own.
	Resources = legacyResources()
	Inputs    = DefaultConfig().Inputs()
)

func legacyResources() []Resource {
	out := make([]Resource, 0, len(content.LegacyOrder))
	for _, k := range content.LegacyOrder {
		out = append(out, Resource(k))
	}
	return out
}
