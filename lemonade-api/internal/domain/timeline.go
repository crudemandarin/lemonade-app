package domain

// The timeline is a compact record of the game so far: a snapshot of capital and
// stock after every meaningful action. The UI draws its two history charts from it,
// and the end-of-game report from Stats. It is derived state, not a rule: nothing in
// the game reads it back.

// PointKind says what produced a timeline point.
type PointKind string

const (
	PointStart   PointKind = "start"
	PointBuy     PointKind = "buy"
	PointSell    PointKind = "sell"
	PointExpand  PointKind = "expand"
	PointUpgrade PointKind = "upgrade"
	// PointSellFacility is a building sold back (Resource set for warehouses).
	PointSellFacility PointKind = "facility_sold"
	PointEndDay       PointKind = "end_day"
)

// TimelinePoint is the state right after one action. Day is the day the action
// happened on (for PointEndDay, the day that just ended).
type TimelinePoint struct {
	Day      int
	Kind     PointKind
	Resource Resource     // buy, sell, and warehouse expansions
	Facility FacilityType // expansions and upgrades
	Qty      int          // cases bought or sold; buildings added
	Amount   int          // dollars: spent (buy, expand, upgrade), earned (sell), or upkeep paid (end of day)
	Produced int          // lemonade made overnight (end of day only)
	Capital  int
	Stock    map[Resource]int // stock after the action
}

// Stats are running totals for the end-of-game report. Unlike the timeline they are
// never compacted, so they stay exact for the whole game.
type Stats struct {
	CasesBought      int
	CasesSold        int // includes stock sold to cover upkeep
	Spent            int // on buying resources
	Earned           int // from selling resources
	FacilitiesBought int
	Upgrades         int
	FacilitySpend    int // expansions and upgrades
	FacilitiesSold   int
	FacilityProceeds int // cash from selling buildings
	Produced         int
	UpkeepPaid       int
	PeakCapital      int
	PeakDay          int
}

const (
	// detailDays is how many recent days keep every buy and sell. Older days keep only
	// milestones (start, facility purchases, end of day), which bounds the payload.
	detailDays = 7
	// maxTimelinePoints is a hard cap, so a player who clicks all day cannot grow it forever.
	maxTimelinePoints = 400
)

// stockSnapshot copies the stock held, leaving out empty commodities so a long
// timeline stays small as the catalog grows. A missing key reads as zero.
func (g *Game) stockSnapshot() map[Resource]int {
	s := make(map[Resource]int, len(g.Inventory))
	for r, n := range g.Inventory {
		if n != 0 {
			s[r] = n
		}
	}
	return s
}

// record appends a point for the game's current state and updates the peak.
func (g *Game) record(p TimelinePoint) {
	p.Capital = g.Capital
	p.Stock = g.stockSnapshot()

	// Repeated clicks on the same trade in the same day read as one event.
	if n := len(g.Timeline); n > 0 && (p.Kind == PointBuy || p.Kind == PointSell) {
		last := &g.Timeline[n-1]
		if last.Kind == p.Kind && last.Day == p.Day && last.Resource == p.Resource {
			last.Qty += p.Qty
			last.Amount += p.Amount
			last.Capital, last.Stock = p.Capital, p.Stock
			g.notePeak()
			return
		}
	}

	g.Timeline = append(g.Timeline, p)
	g.notePeak()
	g.compactTimeline()
}

func (g *Game) notePeak() {
	if g.Capital > g.Stats.PeakCapital {
		g.Stats.PeakCapital, g.Stats.PeakDay = g.Capital, g.Day
	}
}

func isTrade(k PointKind) bool { return k == PointBuy || k == PointSell }

// compactTimeline drops old buys and sells, then trims oldest trades (and, only if
// there are none, oldest milestones) until the cap is met. The start point stays.
func (g *Game) compactTimeline() {
	recentFrom := g.Day - detailDays + 1
	kept := g.Timeline[:0]
	for _, p := range g.Timeline {
		if isTrade(p.Kind) && p.Day < recentFrom {
			continue
		}
		kept = append(kept, p)
	}
	g.Timeline = kept

	for len(g.Timeline) > maxTimelinePoints {
		drop := -1
		for i, p := range g.Timeline {
			if isTrade(p.Kind) {
				drop = i
				break
			}
		}
		if drop < 0 {
			for i, p := range g.Timeline {
				if p.Kind != PointStart {
					drop = i
					break
				}
			}
		}
		if drop < 0 {
			return
		}
		g.Timeline = append(g.Timeline[:drop], g.Timeline[drop+1:]...)
	}
}

// recordFacility notes an expansion or upgrade and adds it to the running totals.
func (g *Game) recordFacility(kind PointKind, f FacilityType, r Resource, qty, cost int) {
	if kind == PointExpand {
		g.Stats.FacilitiesBought++
	} else {
		g.Stats.Upgrades++
	}
	g.Stats.FacilitySpend += cost
	g.record(TimelinePoint{Day: g.Day, Kind: kind, Facility: f, Resource: r, Qty: qty, Amount: cost})
}
