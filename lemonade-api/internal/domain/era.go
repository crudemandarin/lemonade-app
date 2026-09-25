package domain

// Era is the era the player has reached, which gates upgrades (and later recipes).
// Empire A replaces this with the highest territory entered; until then everyone is
// in era 1.
func Era(g Game, cfg Config) int { return 1 }
