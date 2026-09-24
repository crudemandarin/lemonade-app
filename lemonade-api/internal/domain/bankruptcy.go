package domain

// IsBankrupt reports the game-over condition (SPEC rule 16): capital is 0 and
// there is no inventory of any resource. Any inventory is a grace, since every
// resource can be sold at bid.
func IsBankrupt(g Game) bool {
	if g.Capital != 0 {
		return false
	}
	for _, r := range Resources {
		if g.Inventory[r] != 0 {
			return false
		}
	}
	return true
}
