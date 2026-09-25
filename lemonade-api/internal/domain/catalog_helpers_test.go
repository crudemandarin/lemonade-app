package domain

// The tests were written against the fixed list of five commodities. They read the
// same lists from the default catalog, so they keep testing the real game.
var (
	Resources = DefaultConfig().Resources()
	Inputs    = DefaultConfig().Inputs()
)
