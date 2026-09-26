package api

import (
	"errors"
	"math"

	"lemonade-api/internal/domain"
	"lemonade-api/internal/domain/content"
)

// These types mirror lemonade-web/src/app/core/api.models.ts exactly.

type userDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type resourceViewDTO struct {
	Resource      domain.Resource `json:"resource"`
	Stock         int             `json:"stock"`
	Capacity      int             `json:"capacity"`
	Price         int             `json:"price"`
	PreviousPrice *int            `json:"previousPrice"`
	Bid           int             `json:"bid"`
	Ask           int             `json:"ask"`
	History       []int           `json:"history"`
	// Bid and Ask above are the price of the next single case, including the market's
	// reaction to the player's own recent trading. These say how much room is left.
	BuyDepthLeft  int `json:"buyDepthLeft"`
	SellDepthLeft int `json:"sellDepthLeft"`
	// BuyImpactPercent and SellImpactPercent are how far the next case is from the plain quote.
	BuyImpactPercent  int `json:"buyImpactPercent"`
	SellImpactPercent int `json:"sellImpactPercent"`
	// Reach is the depth the market gives this product from every territory held.
	Reach int `json:"reach"`
	// Trade prices the bulk-bar amounts (1, 10, 50, 100, all) with impact, clamped to cash, space and stock.
	Trade tradeLadderDTO `json:"trade"`
	// AvgCost is the average paid per case held (for lemonade, the cost to make one); 0 when none.
	AvgCost int `json:"avgCost"`
	// UnrealizedGain is stock value at the bid minus what it cost; negative is a loss.
	UnrealizedGain int `json:"unrealizedGain"`
	// MovingAverage is the 7-day average price, whole dollars; null without a market analyst.
	MovingAverage *int `json:"movingAverage"`
}

// tradeQuoteDTO is what a trade of some size would cost or raise.
type tradeQuoteDTO struct {
	// Qty is how many cases the quote covers (fewer than asked when cash, space or stock run out).
	Qty   int `json:"qty"`
	Total int `json:"total"`
	// AveragePrice is per case, with impact, to one decimal.
	AveragePrice float64 `json:"averagePrice"`
	// SlippagePercent is how much worse than the plain quote the trade is.
	SlippagePercent float64 `json:"slippagePercent"`
}

// tradeLadderDTO holds a quote per bulk amount, keyed "1", "10", "50", "100" and "all".
type tradeLadderDTO struct {
	Buy  map[string]tradeQuoteDTO `json:"buy"`
	Sell map[string]tradeQuoteDTO `json:"sell"`
}

func toTradeQuote(q domain.TradeQuote) tradeQuoteDTO {
	return tradeQuoteDTO{
		Qty: q.Qty, Total: q.Total,
		AveragePrice:    math.Round(q.Average()*10) / 10,
		SlippagePercent: math.Round(q.Slippage()*1000) / 10,
	}
}

// bulkAmounts are the amounts the trade bar offers; "all" means as many as possible.
var bulkAmounts = []struct {
	key string
	qty int
}{{"1", 1}, {"10", 10}, {"50", 50}, {"100", 100}, {"all", 1_000_000}}

func toTradeLadder(g domain.Game, cfg domain.Config, r domain.Resource) tradeLadderDTO {
	l := tradeLadderDTO{Buy: map[string]tradeQuoteDTO{}, Sell: map[string]tradeQuoteDTO{}}
	for _, a := range bulkAmounts {
		l.Buy[a.key] = toTradeQuote(domain.QuoteBuy(g, cfg, r, a.qty, true))
		l.Sell[a.key] = toTradeQuote(domain.QuoteSell(g, cfg, r, a.qty, true))
	}
	return l
}

func impactPercent(marginal, plain int) int {
	if plain == 0 {
		return 0
	}
	return int(math.Round(100 * (float64(marginal)/float64(plain) - 1)))
}

type upgradeOptionDTO struct {
	TierName        string `json:"tierName"`
	CostPerBuilding int    `json:"costPerBuilding"`
	TotalCost       int    `json:"totalCost"`
	SizePerBuilding int    `json:"sizePerBuilding"`
	// UpkeepIncrease is the extra daily upkeep for the whole type after the upgrade.
	UpkeepIncrease int `json:"upkeepIncrease"`
}

// saleDTO says what selling one building would return and whether it is allowed now.
// Reason is empty when CanSell, else min_facility or stock_exceeds_capacity.
type saleDTO struct {
	SellValue int    `json:"sellValue"`
	CanSell   bool   `json:"canSell"`
	Reason    string `json:"sellBlockedReason"`
	// CasesToSell is how many cases must go first when Reason is stock_exceeds_capacity.
	CasesToSell int `json:"casesToSell"`
}

// warehouseClassDTO is one storage class's building pool: a class holds several
// commodities and its capacity is shared between them.
type warehouseClassDTO struct {
	Class    string            `json:"class"`
	Name     string            `json:"name"`
	Count    int               `json:"count"`
	Capacity int               `json:"capacity"`
	Stock    int               `json:"stock"`
	Members  []domain.Resource `json:"members"`
	saleDTO
}

type warehouseViewDTO struct {
	TierName        string            `json:"tierName"`
	Level           int               `json:"level"`
	MaxLevel        int               `json:"maxLevel"`
	Buildings       int               `json:"buildings"`
	MaxCount        int               `json:"maxCount"`
	SizePerBuilding int               `json:"sizePerBuilding"`
	ExpandCost      int               `json:"expandCost"`
	UpkeepPerDay    int               `json:"upkeepPerDay"`
	Upgrade         *upgradeOptionDTO `json:"upgrade"`
	// MarketDepth is how many cases the market takes at the plain price at this level;
	// UpgradeMarketDepth is the same after the upgrade (0 at max level).
	MarketDepth        int                 `json:"marketDepth"`
	UpgradeMarketDepth int                 `json:"upgradeMarketDepth"`
	Classes            []warehouseClassDTO `json:"classes"`
}

type productionViewDTO struct {
	TierName        string            `json:"tierName"`
	Level           int               `json:"level"`
	MaxLevel        int               `json:"maxLevel"`
	Buildings       int               `json:"buildings"`
	MaxCount        int               `json:"maxCount"`
	SizePerBuilding int               `json:"sizePerBuilding"`
	ExpandCost      int               `json:"expandCost"`
	UpkeepPerDay    int               `json:"upkeepPerDay"`
	Upgrade         *upgradeOptionDTO `json:"upgrade"`
	RatePerDay      int               `json:"ratePerDay"`
	saleDTO
}

type facilitiesDTO struct {
	Warehouse  warehouseViewDTO  `json:"warehouse"`
	Production productionViewDTO `json:"production"`
}

type gameEventDTO struct {
	Key         string                      `json:"key"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Multipliers map[domain.Resource]float64 `json:"multipliers"`
	DaysLeft    int                         `json:"daysLeft"`
}

// commodityDTO is one catalog entry. The view lists them in display order, and every
// per-commodity array in the view (timeline stock, price log prices, base prices)
// follows that order.
type commodityDTO struct {
	Key          domain.Resource `json:"key"`
	Name         string          `json:"name"`
	Category     string          `json:"category"`
	StorageClass string          `json:"storageClass"`
	IsProduct    bool            `json:"isProduct"`
	Order        int             `json:"order"`
}

func toCommodityDTOs(cfg domain.Config) []commodityDTO {
	out := make([]commodityDTO, 0, len(cfg.Commodities))
	for _, c := range cfg.Commodities {
		out = append(out, commodityDTO{
			Key: c.Key, Name: c.Name, Category: c.Category, StorageClass: c.StorageClass,
			IsProduct: c.Product, Order: c.Order,
		})
	}
	return out
}

// inCatalogOrder lists a per-commodity map as an array in catalog order.
func inCatalogOrder(m map[domain.Resource]int, cfg domain.Config) []int {
	out := make([]int, 0, len(cfg.Commodities))
	for _, r := range cfg.Resources() {
		out = append(out, m[r])
	}
	return out
}

// timelinePointDTO is the state right after one action; stock is in catalog order
// (see commodities).
type timelinePointDTO struct {
	Day      int    `json:"day"`
	Kind     string `json:"kind"`
	Resource string `json:"resource,omitempty"`
	Facility string `json:"facility,omitempty"`
	Qty      int    `json:"qty"`
	Amount   int    `json:"amount"`
	Produced int    `json:"produced"`
	Capital  int    `json:"capital"`
	Stock    []int  `json:"stock"`
}

type statsDTO struct {
	CasesBought      int `json:"casesBought"`
	CasesSold        int `json:"casesSold"`
	Spent            int `json:"spent"`
	Earned           int `json:"earned"`
	FacilitiesBought int `json:"facilitiesBought"`
	Upgrades         int `json:"upgrades"`
	FacilitySpend    int `json:"facilitySpend"`
	FacilitiesSold   int `json:"facilitiesSold"`
	FacilityProceeds int `json:"facilityProceeds"`
	Produced         int `json:"produced"`
	UpkeepPaid       int `json:"upkeepPaid"`
	PeakCapital      int `json:"peakCapital"`
	PeakDay          int `json:"peakDay"`
}

// pricePointDTO is one day of the price chart: prices in catalog order, and the
// names of the events active that day.
type pricePointDTO struct {
	Day    int      `json:"day"`
	Prices []int    `json:"prices"`
	Events []string `json:"events"`
}

// netWorthDTO is what the player is worth today: cash, stock at the bid, and what
// every building would resell for. The score of a finished run is its final total.
type netWorthDTO struct {
	Cash       int `json:"cash"`
	Stock      int `json:"stock"`
	Facilities int `json:"facilities"`
	// Acquisitions is what rivals bought out count for (half the price paid).
	Acquisitions int `json:"acquisitions"`
	Total        int `json:"total"`
}

type gameViewDTO struct {
	Day          int                `json:"day"`
	Capital      int                `json:"capital"`
	Status       domain.Status      `json:"status"`
	UpkeepPerDay int                `json:"upkeepPerDay"`
	Resources    []resourceViewDTO  `json:"resources"`
	Facilities   facilitiesDTO      `json:"facilities"`
	Events       []gameEventDTO     `json:"events"`
	Timeline     []timelinePointDTO `json:"timeline"`
	Stats        statsDTO           `json:"stats"`
	Projection   projectionDTO      `json:"projection"`
	PriceLog     []pricePointDTO    `json:"priceLog"`
	NetWorth     netWorthDTO        `json:"netWorth"`
	// RunID names this playthrough; Best is the player's top finished run so far (null if none).
	RunID string   `json:"runId"`
	Best  *bestDTO `json:"best"`
	// BasePrices are the long-run prices in catalog order, for the "% of base" view.
	BasePrices []int `json:"basePrices"`
	// Commodities is the catalog, in display order.
	Commodities []commodityDTO `json:"commodities"`
	// Features are the convenience features turned on by upgrades (for example pnl,
	// repeat_trades); IceKeepCases is what the freezers keep overnight; Forecast is
	// the events the player's upgrades let them see coming.
	Features     []string      `json:"features"`
	IceKeepCases int           `json:"iceKeepCases"`
	Forecast     []forecastDTO `json:"forecast"`
	// Era is the highest territory entered (1 to 5); NextGoal is what to work toward next
	// (null when there is nothing left). The heavy detail is GET /api/game/empire.
	Era      int      `json:"era"`
	EraName  string   `json:"eraName"`
	NextGoal *goalDTO `json:"nextGoal"`
	// Unlocked are the achievements this response's mutation just earned ([] otherwise).
	Unlocked []unlockedDTO `json:"unlocked"`
}

type forecastDTO struct {
	DaysAhead int    `json:"daysAhead"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	Duration  int    `json:"duration"`
}

type pnlDTO struct {
	Sales      int `json:"sales"`
	Purchases  int `json:"purchases"`
	Facilities int `json:"facilities"`
	Upkeep     int `json:"upkeep"`
	IceMade    int `json:"iceMade"`
	Net        int `json:"net"`
}

type priceChangeDTO struct {
	Resource domain.Resource `json:"resource"`
	Before   int             `json:"before"`
	After    int             `json:"after"`
}

type dayReportDTO struct {
	Day        int `json:"day"`
	Produced   int `json:"produced"`
	IceMelted  int `json:"iceMelted"`
	IceKept    int `json:"iceKept"`
	UpkeepPaid int `json:"upkeepPaid"`
	// UpgradeUpkeep is the part of upkeep owed for upgrades; IceMade and IceMadeCost
	// are the ice machine's night; Pnl is the bookkeeper's line (null without one).
	UpgradeUpkeep int     `json:"upgradeUpkeep"`
	IceMade       int     `json:"iceMade"`
	IceMadeCost   int     `json:"iceMadeCost"`
	Pnl           *pnlDTO `json:"pnl"`
	// Stock sold at bid because cash alone could not cover upkeep (0 when none).
	ForcedSaleCases    int              `json:"forcedSaleCases"`
	ForcedSaleProceeds int              `json:"forcedSaleProceeds"`
	CapitalBefore      int              `json:"capitalBefore"`
	CapitalAfter       int              `json:"capitalAfter"`
	PriceChanges       []priceChangeDTO `json:"priceChanges"`
	NewEvents          []gameEventDTO   `json:"newEvents"`
	ExpiredEvents      []gameEventDTO   `json:"expiredEvents"`
	Bankrupt           bool             `json:"bankrupt"`
}

// reportSummaryDTO is one row of the past-days list: enough to scan, without
// the full report.
type reportSummaryDTO struct {
	Day           int      `json:"day"`
	Produced      int      `json:"produced"`
	CapitalBefore int      `json:"capitalBefore"`
	CapitalAfter  int      `json:"capitalAfter"`
	NewEvents     []string `json:"newEvents"`
	ExpiredEvents []string `json:"expiredEvents"`
	Bankrupt      bool     `json:"bankrupt"`
}

func toReportSummary(r domain.DayReport) reportSummaryDTO {
	names := func(events []domain.ActiveEvent) []string {
		out := make([]string, 0, len(events))
		for _, e := range events {
			out = append(out, e.Name)
		}
		return out
	}
	return reportSummaryDTO{
		Day: r.Day, Produced: r.Produced, CapitalBefore: r.CapitalBefore, CapitalAfter: r.CapitalAfter,
		NewEvents: names(r.NewEvents), ExpiredEvents: names(r.ExpiredEvents), Bankrupt: r.Bankrupt,
	}
}

type endDayResponseDTO struct {
	Report dayReportDTO `json:"report"`
	Game   gameViewDTO  `json:"game"`
}

type errorDTO struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// projectionDTO is what End day would do right now: see domain.PreviewEndDay.
type projectionDTO struct {
	LemonadeToProduce int    `json:"lemonadeToProduce"`
	IceToMelt         int    `json:"iceToMelt"`
	IceKept           int    `json:"iceKept"`
	LimitedBy         string `json:"limitedBy"`
}

func toGameView(g domain.Game, cfg domain.Config) gameViewDTO {
	domain.SeedEmpire(&g, cfg)
	quotes := domain.Quotes(g, cfg)

	resources := make([]resourceViewDTO, 0, len(cfg.Commodities))
	for _, r := range cfg.Resources() {
		q := quotes[r]
		m := g.Market[r]
		resources = append(resources, resourceViewDTO{
			Resource:          r,
			Stock:             g.Inventory[r],
			Capacity:          domain.Capacity(g, cfg, r),
			Price:             q.Price,
			PreviousPrice:     m.PreviousEffective,
			Bid:               domain.MarginalBid(g, cfg, r),
			Ask:               domain.MarginalAsk(g, cfg, r),
			BuyDepthLeft:      domain.FreeDepthLeft(g, cfg, r, g.BuyPressure[r]),
			SellDepthLeft:     domain.FreeDepthLeft(g, cfg, r, g.SellPressure[r]),
			BuyImpactPercent:  impactPercent(domain.MarginalAsk(g, cfg, r), q.Ask),
			SellImpactPercent: -impactPercent(domain.MarginalBid(g, cfg, r), q.Bid),
			Reach:             domain.Reach(g, cfg, r),
			Trade:             toTradeLadder(g, cfg, r),
			History:           append([]int{}, m.History...),
			AvgCost:           domain.AvgCost(g, r),
			UnrealizedGain:    domain.UnrealizedGain(g, cfg, r),
			MovingAverage:     movingAverage(g, cfg, m.History),
		})
	}

	return gameViewDTO{
		Day:          g.Day,
		Capital:      g.Capital,
		Status:       g.Status,
		UpkeepPerDay: domain.TotalUpkeep(g, cfg),
		Resources:    resources,
		Facilities: facilitiesDTO{
			Warehouse:  toWarehouseView(g, cfg),
			Production: toProductionView(g, cfg),
		},
		Events:     toEventDTOs(g.Events),
		Timeline:   toTimelineDTOs(g, cfg),
		Stats:      statsDTO(g.Stats),
		Projection: projectionDTO(domain.PreviewEndDay(g, cfg)),
		PriceLog:   toPriceLogDTOs(g, cfg),
		NetWorth:   netWorthDTO(domain.NetWorthBreakdown(g, cfg)),
		RunID:      g.RunID,
		BasePrices: basePrices(cfg),

		Commodities:  toCommodityDTOs(cfg),
		Features:     append([]string{}, domain.Features(g, cfg)...),
		IceKeepCases: domain.IceKeep(g, cfg),
		Forecast:     toForecastDTOs(domain.Forecast(g, cfg)),
		Era:          domain.Era(g, cfg),
		EraName:      domain.EraName(g, cfg),
		NextGoal:     toGoal(g, cfg),
		Unlocked:     []unlockedDTO{},
	}
}

func toForecastDTOs(f []domain.ForecastEntry) []forecastDTO {
	out := make([]forecastDTO, 0, len(f))
	for _, e := range f {
		out = append(out, forecastDTO{DaysAhead: e.DaysAhead, Key: e.Key, Name: e.Name, Duration: e.Duration})
	}
	return out
}

// movingAverage is the mean of the last 7 daily prices, for a player with a market
// analyst; nil for everyone else.
func movingAverage(g domain.Game, cfg domain.Config, history []int) *int {
	if !domain.HasFeature(g, cfg, "moving_average") || len(history) == 0 {
		return nil
	}
	recent := history[max(0, len(history)-7):]
	sum := 0
	for _, p := range recent {
		sum += p
	}
	avg := int(math.Round(float64(sum) / float64(len(recent))))
	return &avg
}

// toTimelineDTOs maps the timeline. A game saved before the timeline existed has none,
// so it gets a start point from its current state and the charts always have something.
func toTimelineDTOs(g domain.Game, cfg domain.Config) []timelinePointDTO {
	if len(g.Timeline) == 0 {
		stock := inCatalogOrder(g.Inventory, cfg)
		return []timelinePointDTO{{Day: g.Day, Kind: string(domain.PointStart), Capital: g.Capital, Stock: stock}}
	}
	return timelinePointDTOs(g.Timeline, cfg)
}

// timelinePointDTOs maps timeline points, for a live game or a finished run.
func timelinePointDTOs(points []domain.TimelinePoint, cfg domain.Config) []timelinePointDTO {
	out := make([]timelinePointDTO, 0, len(points))
	for _, p := range points {
		out = append(out, timelinePointDTO{
			Day: p.Day, Kind: string(p.Kind), Resource: string(p.Resource), Facility: string(p.Facility),
			Qty: p.Qty, Amount: p.Amount, Produced: p.Produced, Capital: p.Capital, Stock: inCatalogOrder(p.Stock, cfg),
		})
	}
	return out
}

func basePrices(cfg domain.Config) []int {
	return inCatalogOrder(cfg.BasePrice, cfg)
}

// toPriceLogDTOs maps the price log, turning event keys into display names.
func toPriceLogDTOs(g domain.Game, cfg domain.Config) []pricePointDTO {
	return priceLogDTOs(g.PriceLog, cfg)
}

func priceLogDTOs(log []domain.PricePoint, cfg domain.Config) []pricePointDTO {
	names := make(map[string]string, len(cfg.Events))
	for _, e := range cfg.Events {
		names[e.Key] = e.Name
	}
	out := make([]pricePointDTO, 0, len(log))
	for _, p := range log {
		events := make([]string, 0, len(p.Events))
		for _, key := range p.Events {
			if name, ok := names[key]; ok {
				events = append(events, name)
			} else {
				events = append(events, key)
			}
		}
		out = append(out, pricePointDTO{Day: p.Day, Prices: inCatalogOrder(p.Prices, cfg), Events: events})
	}
	return out
}

func toSale(g domain.Game, cfg domain.Config, kind domain.FacilityType, r domain.Resource) saleDTO {
	value := domain.ResaleValue(cfg, cfg.ProductionTiers[g.ProductionLevel-1])
	if kind == domain.Warehouse {
		value = domain.ResaleValue(cfg, cfg.WarehouseTiers[g.WarehouseLevel-1])
	}
	s := saleDTO{SellValue: value, CanSell: true}
	var excess *domain.StockExceedsCapacityError
	switch err := domain.CanSellFacility(g, cfg, kind, r); {
	case err == nil:
	case errors.As(err, &excess):
		s.CanSell, s.Reason, s.CasesToSell = false, "stock_exceeds_capacity", excess.Excess
	default:
		s.CanSell, s.Reason = false, "min_facility"
	}
	return s
}

func toWarehouseView(g domain.Game, cfg domain.Config) warehouseViewDTO {
	tier := cfg.WarehouseTiers[g.WarehouseLevel-1]

	buildings := 0
	classes := make([]warehouseClassDTO, 0, 4)
	for _, class := range domain.StorageClasses(cfg) {
		members := domain.ClassMembers(cfg, class)
		buildings += g.WarehouseQty[class]
		classes = append(classes, warehouseClassDTO{
			Class:    class,
			Name:     content.StorageClassNames[class],
			Count:    g.WarehouseQty[class],
			Capacity: domain.ClassCapacity(g, cfg, class),
			Stock:    domain.ClassStock(g, cfg, class),
			Members:  members,
			saleDTO:  toSale(g, cfg, domain.Warehouse, domain.Resource(class)),
		})
	}

	return warehouseViewDTO{
		TierName:           tier.Name,
		Level:              g.WarehouseLevel,
		MaxLevel:           domain.LevelCap(g, cfg),
		Buildings:          buildings,
		MaxCount:           domain.BuildingCap(g, cfg),
		SizePerBuilding:    tier.Size,
		ExpandCost:         tier.BuildCost,
		UpkeepPerDay:       domain.WarehouseUpkeep(g, cfg),
		Upgrade:            upgradeOption(cfg.WarehouseTiers, g.WarehouseLevel, domain.LevelCap(g, cfg), buildings),
		MarketDepth:        domain.Reach(g, cfg, domain.Lemonade),
		UpgradeMarketDepth: upgradeDepth(g, cfg),
		Classes:            classes,
	}
}

func toProductionView(g domain.Game, cfg domain.Config) productionViewDTO {
	tier := cfg.ProductionTiers[g.ProductionLevel-1]
	return productionViewDTO{
		TierName:        tier.Name,
		Level:           g.ProductionLevel,
		MaxLevel:        domain.LevelCap(g, cfg),
		Buildings:       g.ProductionQty,
		MaxCount:        domain.BuildingCap(g, cfg),
		SizePerBuilding: tier.Size,
		ExpandCost:      tier.BuildCost,
		UpkeepPerDay:    domain.ProductionUpkeep(g, cfg),
		Upgrade:         upgradeOption(cfg.ProductionTiers, g.ProductionLevel, domain.LevelCap(g, cfg), g.ProductionQty),
		RatePerDay:      domain.ProductionCapacity(g, cfg),
		saleDTO:         toSale(g, cfg, domain.Production, ""),
	}
}

func upgradeDepth(g domain.Game, cfg domain.Config) int {
	if g.WarehouseLevel >= domain.LevelCap(g, cfg) {
		return 0
	}
	return domain.ReachAtLevel(g, cfg, domain.Lemonade, g.WarehouseLevel+1)
}

// upgradeOption describes the next level, or nil at max level.
func upgradeOption(tiers []domain.Tier, level, maxLevel, buildings int) *upgradeOptionDTO {
	if level >= maxLevel {
		return nil
	}
	current, next := tiers[level-1], tiers[level]
	return &upgradeOptionDTO{
		TierName:        next.Name,
		CostPerBuilding: current.UpgradeCost,
		TotalCost:       current.UpgradeCost * buildings,
		SizePerBuilding: next.Size,
		UpkeepIncrease:  (next.Upkeep - current.Upkeep) * buildings,
	}
}

func toEventDTOs(events []domain.ActiveEvent) []gameEventDTO {
	out := make([]gameEventDTO, 0, len(events))
	for _, e := range events {
		out = append(out, gameEventDTO{
			Key:         e.Key,
			Name:        e.Name,
			Description: e.Description,
			Multipliers: e.Multipliers,
			DaysLeft:    e.DaysLeft,
		})
	}
	return out
}

func toDayReport(r domain.DayReport) dayReportDTO {
	changes := make([]priceChangeDTO, 0, len(r.PriceChanges))
	for _, c := range r.PriceChanges {
		changes = append(changes, priceChangeDTO{Resource: c.Resource, Before: c.Before, After: c.After})
	}
	return dayReportDTO{
		Day:                r.Day,
		Produced:           r.Produced,
		IceMelted:          r.IceMelted,
		IceKept:            r.IceKept,
		UpkeepPaid:         r.UpkeepPaid,
		UpgradeUpkeep:      r.UpgradeUpkeep,
		IceMade:            r.IceMade,
		IceMadeCost:        r.IceMadeCost,
		Pnl:                toPnl(r.Pnl),
		ForcedSaleCases:    r.ForcedSaleCases,
		ForcedSaleProceeds: r.ForcedSaleProceeds,
		CapitalBefore:      r.CapitalBefore,
		CapitalAfter:       r.CapitalAfter,
		PriceChanges:       changes,
		NewEvents:          toEventDTOs(r.NewEvents),
		ExpiredEvents:      toEventDTOs(r.ExpiredEvents),
		Bankrupt:           r.Bankrupt,
	}
}

func toPnl(p *domain.PnlLine) *pnlDTO {
	if p == nil {
		return nil
	}
	d := pnlDTO(*p)
	return &d
}
