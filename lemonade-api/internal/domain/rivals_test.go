package domain

import (
	"errors"
	"math"
	"testing"
)

// shareTotal is the player's share plus every active rival's share and any folded
// rival's share still waiting to be handed out.
func shareTotal(g Game, cfg Config, territory string) float64 {
	total := g.Territories[territory].Share
	for _, d := range cfg.Rivals {
		if d.Territory == territory {
			if r, ok := g.Rivals[d.Key]; ok && r.Status != RivalAcquired {
				total += r.Share
			}
		}
	}
	return total
}

func rich(g *Game) { g.Capital = 500_000_000 }

func enterAll(t *testing.T, g *Game, cfg Config, keys ...string) {
	t.Helper()
	rich(g)
	for _, k := range keys {
		if err := EnterTerritory(g, cfg, k); err != nil {
			t.Fatalf("enter %s: %v", k, err)
		}
	}
}

func TestEnteringATerritory(t *testing.T) {
	g, cfg := newTestGame()
	if err := EnterTerritory(&g, cfg, "city"); !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("with $1000: %v", err)
	}
	g.Capital = 20000
	if err := EnterTerritory(&g, cfg, "region"); !errors.Is(err, ErrTerritoryLocked) {
		t.Fatalf("skipping the city: %v", err)
	}
	if err := EnterTerritory(&g, cfg, "nowhere"); !errors.Is(err, ErrUnknownTerritory) {
		t.Fatalf("unknown: %v", err)
	}
	if err := EnterTerritory(&g, cfg, "city"); err != nil {
		t.Fatal(err)
	}
	if g.Capital != 5000 || g.Territories["city"].Share != 10 || Era(g, cfg) != 2 {
		t.Fatalf("after entering: cash %d, city %+v, era %d", g.Capital, g.Territories["city"], Era(g, cfg))
	}
	if len(activeRivals(g, cfg, "city")) != 4 {
		t.Fatal("the city's four rivals appear")
	}
	if err := EnterTerritory(&g, cfg, "city"); !errors.Is(err, ErrAlreadyEntered) {
		t.Fatalf("twice: %v", err)
	}
	if shareTotal(g, cfg, "city") != 100 {
		t.Fatalf("city shares add up to %v", shareTotal(g, cfg, "city"))
	}
}

func TestBuyoutAccounting(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 10_000
	nwBefore := NetWorth(g, cfg)
	price, _ := BuyoutPrice(g, cfg, "lil_lucy", false)
	if price != int(math.Round(2500*cfg.ValuationMultiple)) { // Lucy's premium is 1.0
		t.Fatalf("Lucy costs %d", price)
	}
	if err := BuyOut(&g, cfg, "lil_lucy", false); err != nil {
		t.Fatal(err)
	}
	if g.Capital != 10_000-price {
		t.Fatalf("cash %d", g.Capital)
	}
	if got := g.Territories["neighborhood"].Share; got != 55 {
		t.Fatalf("share %v, want 55", got)
	}
	if r := g.Rivals["lil_lucy"]; r.Status != RivalAcquired || r.Share != 0 || r.PricePaid != price {
		t.Fatalf("rival %+v", r)
	}
	if shareTotal(g, cfg, "neighborhood") != 100 {
		t.Fatal("shares no longer add up")
	}
	if got := AcquisitionValue(g, cfg); got != price/2 {
		t.Fatalf("acquisition value %d, want half of %d", got, price)
	}
	if got := NetWorth(g, cfg) - nwBefore; got != -price+price/2 {
		t.Fatalf("net worth moved by %d, want the price lost less what the acquisition counts for", got)
	}
	if freeDepth(g, cfg, Lemonade) != 110 { // 80 * 55 / 40
		t.Fatalf("reach %d, want 110", freeDepth(g, cfg, Lemonade))
	}
	if err := BuyOut(&g, cfg, "lil_lucy", false); !errors.Is(err, ErrRivalGone) {
		t.Fatalf("second buyout: %v", err)
	}
	if err := BuyOut(&g, cfg, "city_nobody", false); !errors.Is(err, ErrUnknownRival) {
		t.Fatalf("unknown: %v", err)
	}
}

func TestBuyoutNeedsCash(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 100
	if err := BuyOut(&g, cfg, "lil_lucy", false); !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("%v", err)
	}
}

func TestSourSamRefusesUntilTheLeaderHasSixtyPercent(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	var refuse *RivalRefusesError
	if err := BuyOut(&g, cfg, "sour_sam", false); !errors.As(err, &refuse) || refuse.NeedShare != 60 {
		t.Fatalf("friendly buyout: %v", err)
	}
	friendly, _ := BuyoutPrice(g, cfg, "sour_sam", false)
	hostile, _ := BuyoutPrice(g, cfg, "sour_sam", true)
	if hostile <= friendly || float64(hostile) != math.Round(g.Rivals["sour_sam"].Valuation*1.6) {
		t.Fatalf("hostile %d vs friendly %d", hostile, friendly)
	}
	if err := BuyOut(&g, cfg, "sour_sam", true); err != nil {
		t.Fatalf("hostile: %v", err)
	}
	if !g.Rivals["sour_sam"].Hostile {
		t.Fatal("the takeover is recorded as hostile")
	}
	// Rivals of a territory not yet entered are not in the game.
	if err := BuyOut(&g, cfg, "lemonopoly", false); !errors.Is(err, ErrUnknownRival) {
		t.Fatalf("a rival of a territory not entered: %v", err)
	}
}

func TestMergerOfferDiscountsTheBuyout(t *testing.T) {
	g, cfg := newTestGame()
	full, _ := BuyoutPrice(g, cfg, "squeeze_box", false)
	r := g.Rivals["squeeze_box"]
	r.OfferDaysLeft = 2
	g.Rivals["squeeze_box"] = r
	got, offer := BuyoutPrice(g, cfg, "squeeze_box", false)
	if !offer || got != int(math.Round(float64(full)*0.8)) {
		t.Fatalf("offer price %d (offer=%v), full %d", got, offer, full)
	}
}

func TestCampaign(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	cost := CampaignCost(cfg, "neighborhood", 1)
	before := g.Capital
	if err := RunCampaign(&g, cfg, "neighborhood", 1); err != nil {
		t.Fatal(err)
	}
	if g.Capital != before-cost || g.Territories["neighborhood"].CampaignDaysLeft != 5 || g.Territories["neighborhood"].CampaignBonus != 0.05 {
		t.Fatalf("after: cash %d, %+v", g.Capital, g.Territories["neighborhood"])
	}
	if err := RunCampaign(&g, cfg, "neighborhood", 2); !errors.Is(err, ErrCampaignRunning) {
		t.Fatalf("second: %v", err)
	}
	if err := RunCampaign(&g, cfg, "city", 1); !errors.Is(err, ErrNotEntered) {
		t.Fatalf("not entered: %v", err)
	}
	if err := RunCampaign(&g, cfg, "neighborhood", 9); !errors.Is(err, ErrInvalidCampaign) {
		t.Fatalf("bad level: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := EndDay(&g, cfg); err != nil {
			t.Fatal(err)
		}
		rich(&g)
	}
	if g.Territories["neighborhood"].CampaignDaysLeft != 0 || g.Territories["neighborhood"].CampaignBonus != 0 {
		t.Fatalf("the campaign should have ended: %+v", g.Territories["neighborhood"])
	}
}

// A passive player at the start share keeps it: rivals push, but the floor holds, so the
// phase 0 game is unchanged for anyone who never touches the empire.
func TestAPassivePlayerKeepsTheNeighborhood(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	for i := 0; i < 60; i++ {
		if _, err := EndDay(&g, cfg); err != nil {
			t.Fatal(err)
		}
		rich(&g)
		if s := g.Territories["neighborhood"].Share; s != 40 {
			t.Fatalf("day %d: share %v", g.Day, s)
		}
	}
}

func TestRivalsPushAnUnderdefendedTerritoryDownToTheFloorNoFurther(t *testing.T) {
	g, cfg := newTestGame()
	enterAll(t, &g, cfg, "city")
	prev := g.Territories["city"].Share
	for i := 0; i < 40; i++ {
		if _, err := EndDay(&g, cfg); err != nil {
			t.Fatal(err)
		}
		rich(&g)
		s := g.Territories["city"].Share
		if prev-s > cfg.MaxShareShiftPerDay+1e-9 || s-prev > cfg.MaxShareShiftPerDay+1e-9 {
			t.Fatalf("day %d: moved %v points", g.Day, s-prev)
		}
		if s < 5-1e-9 {
			t.Fatalf("day %d: share %v is under the floor (half of 10)", g.Day, s)
		}
		prev = s
	}
	if prev > 6 {
		t.Fatalf("an undefended city stayed at %v, expected it to sink toward 5", prev)
	}
}

func TestACampaignHoldsAndGainsShare(t *testing.T) {
	g, cfg := newTestGame()
	enterAll(t, &g, cfg, "city")
	start := g.Territories["city"].Share
	for i := 0; i < 20; i++ {
		g.SellPressure[Lemonade] = 400 // sells well: full fill
		if g.Territories["city"].CampaignDaysLeft == 0 {
			if err := RunCampaign(&g, cfg, "city", 3); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := EndDay(&g, cfg); err != nil {
			t.Fatal(err)
		}
		rich(&g)
	}
	if g.Territories["city"].Share <= start+3 {
		t.Fatalf("a constant ad blitz moved the share from %v to only %v", start, g.Territories["city"].Share)
	}
}

// Whatever the player does, every territory's shares stay at 100 and no share is negative.
func TestSharesAlwaysAddUpTo100(t *testing.T) {
	for seed := int64(1); seed <= 6; seed++ {
		cfg := DefaultConfig()
		g := NewGame(cfg, seed)
		enterAll(t, &g, cfg, "city", "region")
		for day := 0; day < 120; day++ {
			g.SellPressure[Lemonade] = float64(day%7) * 60
			switch day % 9 {
			case 0:
				_ = RunCampaign(&g, cfg, "city", 2)
			case 3:
				_ = BuyOut(&g, cfg, "big_pour", false)
			case 5:
				_ = BuyOut(&g, cfg, "lil_lucy", day%2 == 0)
			}
			if _, err := EndDay(&g, cfg); err != nil {
				t.Fatal(err)
			}
			rich(&g)
			for _, d := range cfg.Territories {
				if !g.Territories[d.Key].Entered {
					continue
				}
				if got := shareTotal(g, cfg, d.Key); math.Abs(got-100) > 1e-6 {
					t.Fatalf("seed %d day %d %s: shares add up to %v", seed, g.Day, d.Key, got)
				}
				if g.Territories[d.Key].Share < -1e-9 {
					t.Fatalf("negative player share in %s", d.Key)
				}
			}
			for k, r := range g.Rivals {
				if r.Share < -1e-9 {
					t.Fatalf("seed %d day %d: %s has share %v", seed, g.Day, k, r.Share)
				}
			}
		}
	}
}

// Adding rivals (and the empire) never changes the market: the same seed walks the same
// prices whether or not the player has entered territories or the rivals act.
func TestRivalsNeverChangeMarketPrices(t *testing.T) {
	cfg := DefaultConfig()
	plain := NewGame(cfg, 42)
	busy := NewGame(cfg, 42)
	enterAll(t, &busy, cfg, "city", "region")
	for day := 0; day < 80; day++ {
		rich(&plain)
		rich(&busy)
		if _, err := EndDay(&plain, cfg); err != nil {
			t.Fatal(err)
		}
		if _, err := EndDay(&busy, cfg); err != nil {
			t.Fatal(err)
		}
		for _, r := range cfg.Resources() {
			if plain.Market[r].Price != busy.Market[r].Price {
				t.Fatalf("day %d %s: walked price %v vs %v", plain.Day, r, plain.Market[r].Price, busy.Market[r].Price)
			}
		}
	}
	// And the same game twice is the same game.
	a, b := NewGame(cfg, 7), NewGame(cfg, 7)
	enterAll(t, &a, cfg, "city")
	enterAll(t, &b, cfg, "city")
	for i := 0; i < 60; i++ {
		_, _ = EndDay(&a, cfg)
		_, _ = EndDay(&b, cfg)
		rich(&a)
		rich(&b)
	}
	if a.Rivals["zest_express"] != b.Rivals["zest_express"] || a.Territories["city"] != b.Territories["city"] {
		t.Fatal("rivals are not deterministic")
	}
}

// A rival does what it telegraphed: the announcement names the day and the event, and
// the price war then appears as an ordinary event on exactly that day.
func TestARivalDoesWhatItTelegraphed(t *testing.T) {
	cfg := DefaultConfig()
	seen := 0
	for seed := int64(1); seed <= 40 && seen < 3; seed++ {
		g := NewGame(cfg, seed)
		enterAll(t, &g, cfg, "city")
		// Make the player a threat so rivals fight back.
		city := g.Territories["city"]
		city.Share = 30
		g.Territories["city"] = city
		for day := 0; day < 80; day++ {
			before := g.Clone()
			rich(&g)
			if _, err := EndDay(&g, cfg); err != nil {
				t.Fatal(err)
			}
			for key, r := range before.Rivals {
				if r.TelegraphKey == EventPriceWar && r.TelegraphDay == g.Day && r.Status == RivalActive {
					seen++
					found := false
					for _, e := range g.Events {
						if e.Key == EventPriceWar && e.Multipliers[Lemonade] == cfg.PriceWarMultiplier {
							found = true
						}
					}
					if !found && !priceWarActive(before) {
						t.Fatalf("seed %d: %s announced a price war for day %d and it did not happen", seed, key, g.Day)
					}
				}
			}
			if g.Status != StatusActive {
				break
			}
		}
	}
	if seen == 0 {
		t.Fatal("no price war was ever telegraphed in 40 seeds: the rival never fights back")
	}
}

func TestPriceWarsAreOrdinaryEventsAndNeverTouchStock(t *testing.T) {
	g, cfg := newTestGame()
	enterAll(t, &g, cfg, "city")
	g.Inventory[Lemonade] = 30
	g.Rivals["zest_express"] = func() RivalState {
		r := g.Rivals["zest_express"]
		r.TelegraphKey, r.TelegraphDay = EventPriceWar, g.Day+1
		return r
	}()
	walked := g.Market[Lemonade].Price
	_ = walked
	if _, err := EndDay(&g, cfg); err != nil {
		t.Fatal(err)
	}
	if !priceWarActive(g) {
		t.Fatal("the announced price war did not start")
	}
	if g.Inventory[Lemonade] < 0 {
		t.Fatal("stock destroyed")
	}
}

func TestOpportunistOffersAMergerAfterLosingShare(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	// Give the player a strong presence so rivals keep losing share.
	for i := 0; i < 30; i++ {
		g.SellPressure[Lemonade] = 500
		if g.Territories["neighborhood"].CampaignDaysLeft == 0 {
			_ = RunCampaign(&g, cfg, "neighborhood", 3)
		}
		_, _ = EndDay(&g, cfg)
		rich(&g)
		if g.Rivals["squeeze_box"].OfferDaysLeft > 0 {
			return
		}
	}
	t.Fatalf("no merger offer after a month of losing share: %+v", g.Rivals["squeeze_box"])
}

func TestASqueezedRivalFoldsAndItsShareIsHandedOut(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	r := g.Rivals["squeeze_box"]
	r.Valuation = 100 // far under the fold threshold
	g.Rivals["squeeze_box"] = r
	folded := false
	for i := 0; i < 20; i++ {
		_, _ = EndDay(&g, cfg)
		rich(&g)
		if g.Rivals["squeeze_box"].Status == RivalFolded {
			folded = true
		}
		if math.Abs(shareTotal(g, cfg, "neighborhood")-100) > 1e-6 {
			t.Fatalf("day %d: shares add up to %v", g.Day, shareTotal(g, cfg, "neighborhood"))
		}
	}
	if !folded {
		t.Fatal("the rival never folded")
	}
	if got := g.Rivals["squeeze_box"].Share; got > 1e-9 {
		t.Fatalf("freed share left over: %v", got)
	}
}

func TestIntegratedRivalsCutInputDepthUntilBoughtOut(t *testing.T) {
	g, cfg := newTestGame()
	enterAll(t, &g, cfg, "city", "region")
	with := freeDepth(g, cfg, Lemon)
	other := freeDepth(g, cfg, Sugar)
	if with >= other {
		t.Fatalf("lemon depth %d should be below sugar depth %d while Golden Grove is active", with, other)
	}
	if err := BuyOut(&g, cfg, "golden_grove", false); err != nil {
		t.Fatal(err)
	}
	if got := freeDepth(g, cfg, Lemon); got <= other {
		t.Fatalf("lemon depth %d should beat sugar depth %d after buying Golden Grove (+20%%)", got, other)
	}
}
