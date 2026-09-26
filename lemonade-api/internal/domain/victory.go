package domain

// Victory (late game Empire B): the run is won when the player holds at least half of
// the last territory, or has bought out its megacorp. Winning changes nothing about
// play: the run keeps going, its score keeps growing, and the win day stays recorded.

// VictoryShare is the percent of the World that wins the game.
const VictoryShare = 50

// FinalTerritory is the last territory of the ladder; FinalRival the rival that owns
// a buyout win.
const FinalRival = "global_citrus"

// VictoryConditionMet reports whether the game meets the victory condition now.
func VictoryConditionMet(g Game, cfg Config) bool {
	if len(cfg.Territories) == 0 {
		return false
	}
	last := cfg.Territories[len(cfg.Territories)-1]
	if t := g.Territories[last.Key]; t.Entered && t.Share >= VictoryShare {
		return true
	}
	return g.Rivals[FinalRival].Status == RivalAcquired
}

// WonByBuyout reports whether the final rival was bought (the "underdog" achievement
// is the win without it).
func WonByBuyout(g Game) bool { return g.Rivals[FinalRival].Status == RivalAcquired }

// RecordVictory notes the first day the condition is met and the net worth then. It
// records once, never for a run that is not active, and changes nothing the rules read.
// The API layer calls it after every mutation, inside the same transaction.
func RecordVictory(g *Game, cfg Config) {
	if g.Status != StatusActive || g.WonOnDay > 0 || !VictoryConditionMet(*g, cfg) {
		return
	}
	g.WonOnDay = g.Day
	g.WonNetWorth = NetWorth(*g, cfg)
}
