# Late game content catalog (draft 1, for review)

Companion to `LATE-GAME-DESIGN.md`. Everything here is **placeholder content to react to**: names, numbers and effects will be cut, merged and retuned by simulation. Every table is written so it maps one-to-one onto a `Config` table: one row is one entry, and keys are stable snake_case.

Money is in whole dollars. "Era" means the earliest era where the entry appears (1 Neighborhood, 2 City, 3 Region, 4 Nation, 5 World).

---

## A. Territories

| # | Key | Name | Lemonade demand depth (cases/day) | Player share on entry | Entry cost | Hub upkeep/day | Unlocks |
|---|---|---|---|---|---|---|---|
| 1 | `neighborhood` | The Neighborhood | 200 | 40% (start; equals today's 80 free cases) | none | $0 | production and warehouse tiers 1 and 2, cap 10 buildings per type |
| 2 | `city` | Citrus City | 800 | 10% | $15,000 | $150 | tier 3, +10 cap, managers, era 2 recipes |
| 3 | `region` | Sunbelt Region | 3,000 | 8% | $120,000 | $900 | tier 4, +10 cap, bottled products, supplier contracts |
| 4 | `nation` | The Nation | 12,000 | 5% | $1,000,000 | $5,000 | tier 5, +10 cap, premium recipes, fast-forward |
| 5 | `world` | The World | 50,000 | 3% | $8,000,000 | $30,000 | tiers 6 and 7, +20 cap, global rivals, victory condition |

Hub synergy: each territory owned beyond the first cuts all hub upkeep by 5% (up to 20%).

**Product demand depth** is a share of the territory's lemonade depth (table H, column "Depth vs lemonade"). A World-era business with eight products has about four times the depth of lemonade alone.

---

## B. Facility tier ladder (flipped to economies of scale)

Per building. Production "size" is cases produced per day; warehouse "size" is cases stored per storage class.

| Level | Production tier | Size | Build | Upkeep | $/case upkeep | Warehouse tier | Size | Build | Upkeep | Era |
|---|---|---|---|---|---|---|---|---|---|---|
| 1 | Kitchen | 10 | $500 | $20 | 2.00 | Pantry | 10 | $100 | $2 | 1 |
| 2 | Food Truck | 25 | $1,100 | $40 | 1.60 | Garage | 25 | $220 | $4 | 1 |
| 3 | Bottling Plant | 60 | $2,200 | $75 | 1.25 | Barn | 60 | $450 | $8 | 2 |
| 4 | Lemonade Factory | 150 | $4,500 | $150 | 1.00 | Industrial Warehouse | 150 | $900 | $15 | 3 |
| 5 | Regional Plant | 400 | $10,000 | $320 | 0.80 | Distribution Center | 400 | $2,000 | $32 | 4 |
| 6 | Mega Plant | 1,000 | $22,000 | $650 | 0.65 | Logistics Hub | 1,000 | $4,500 | $65 | 5 |
| 7 | Global Network | 2,500 | $48,000 | $1,300 | 0.52 | Automated Megastore | 2,500 | $10,000 | $130 | 5 |

Upgrade cost from level L to L+1 is about 60% of the next tier's build cost per building, so upgrading is cheaper than rebuilding. Keep a single shared level per facility type (current rule). Existing save games on levels 3 and 4 are **grandfathered**: they keep their level even though they are in era 1.

---

## C. Rivals

Starting shares within each territory add up to 100% minus the player's entry share. "Buyout" is the rough valuation at the start of the era; it grows over time. Personality codes: **P** passive, **A** aggressive, **Pr** premium, **O** opportunist, **I** integrated.

### Neighborhood (60%)
| Key | Name | Share | Pers. | Buyout | Flavor and rule |
|---|---|---|---|---|---|
| `lil_lucy` | Lil' Lucy's Lemonade | 15% | P | $2,500 | A kid's stand. She sells to you gladly: the friendly premium is only 1.0x. |
| `sour_sam` | Sour Sam's Stand | 25% | A | $6,000 | Starts price wars on holidays. Refuses friendly buyouts until you hold 60%. |
| `squeeze_box` | The Squeeze Box | 20% | O | $4,500 | Offers a merger at a 20% discount when it has lost share 5 days running. |

### Citrus City (90%)
| Key | Name | Share | Pers. | Buyout | Flavor and rule |
|---|---|---|---|---|---|
| `main_st_squeeze` | Main Street Squeeze | 30% | P | $60,000 | The established local chain; slow and steady. |
| `cafe_citron` | Café Citron | 15% | Pr | $55,000 | Premium café; immune to price wars; owning it gives +5% depth for premium recipes in City. |
| `zest_express` | Zest Express | 25% | A | $50,000 | Fleet of food trucks; campaigns against the share leader. |
| `pucker_up` | Pucker Up Co. | 20% | O | $40,000 | Social-media darling; share swings with "Influencer" events. |

### Sunbelt Region (92%)
| Key | Name | Share | Pers. | Buyout | Flavor and rule |
|---|---|---|---|---|---|
| `sunbelt_bev` | Sunbelt Beverages | 30% | P | $600,000 | Regional bottler; acquiring it gives 4 Bottling Plants at 50% off. |
| `golden_grove` | Golden Grove Co-op | 20% | I | $450,000 | Owns orchards; while active, lemon input depth is 20% lower for you. Buying it reverses that (+20%). |
| `valley_fresh` | Valley Fresh | 17% | Pr | $400,000 | Organic brand; holds share stubbornly. |
| `tart_and_co` | Tart & Co. | 15% | A | $300,000 | Aggressive discounter. |
| `big_pour` | Big Pour Regional | 10% | O | $220,000 | Overextended; likely to fold (then its share is split, and you win it by presence). |

### The Nation (95%)
| Key | Name | Share | Pers. | Buyout | Flavor and rule |
|---|---|---|---|---|---|
| `national_lemon_works` | National Lemon Works | 30% | P | $5M | The old giant. |
| `fizz_nation` | FizzNation | 20% | A | $3.5M | Bottled-sparkling specialist; runs price wars on sparkling lemonade. |
| `yellowbrand` | Yellowbrand | 20% | Pr | $4M | Household name; +10% presence in every territory while active. |
| `squeezo` | Squeezo | 15% | O | $2.5M | Tech-y delivery app; makes merger offers. |
| `harvest_holdings` | Harvest Holdings | 10% | I | $2M | Controls sugar contracts; sugar input depth −20% while active. |

### The World (97%)
| Key | Name | Share | Pers. | Buyout | Flavor and rule |
|---|---|---|---|---|---|
| `global_citrus` | Global Citrus Holdings | 35% | P | $60M | The final boss. Buying it is the victory condition's alternative path. |
| `megapour` | MegaPour International | 25% | A | $40M | Global price wars; telegraphs 2 days ahead. |
| `lemonopoly` | Lemonopoly Group | 22% | Pr | $45M | Refuses buyout until you own 25% of the World. |
| `zestcorp` | Zestcorp | 15% | I | $25M | Controls citrus supply; lemon and lime input depth −15% while active. |

---

## D. Rival events (from the rival tick; never touch inventory)

| Key | Trigger | Effect | Telegraph |
|---|---|---|---|
| `price_war` | an aggressive rival, at most once per 20 days per rival | the named product's price ×0.85 in that territory's share of depth for 3 days | "Sour Sam is slashing prices tomorrow" |
| `rival_campaign` | aggressive rival vs the share leader | rival presence +20% for 5 days | 1 day ahead |
| `merger_offer` | opportunist losing share | buyout at 20% off for 3 days (a popup, with no obligation) | none (it's an offer) |
| `rival_folds` | a rival's valuation below a threshold | its share is freed; over 5 days it goes to whoever has more presence | 3 days of "struggling" |
| `rival_expands` | passive rival, long streak of gains | +2 share points for the rival | 1 day ahead |
| `hostile_bid` | you hold at least 50% of a territory | a rival tries to lure your share (presence contest for 3 days) | 1 day ahead |
| `supply_squeeze` | integrated rival active | an input's depth −20% for 3 days | 1 day ahead |
| `poach_manager` | late eras, you have managers | one manager is idle for 1 day unless the "Retention bonus" upgrade is owned | 1 day ahead |

---

## E. Upgrades (about 40)

"Upkeep" is per day. Effects are typed modifiers (design doc section 6.4). Costs scale by era.

### Freshness and storage
| Key | Name | Era | Cost | Upkeep | Effect |
|---|---|---|---|---|---|
| `freezer_1` | Chest freezer | 1 | $1,200 | $5 | Keep up to 20 cases of ice one extra night |
| `freezer_2` | Walk-in freezer | 2 | $8,000 | $20 | Keep up to 150 ice |
| `freezer_3` | Cold storage wing | 3 | $60,000 | $120 | Keep up to 1,200 ice |
| `cold_room` | Cold room | 2 | $6,000 | $15 | Fresh fruit and herbs keep 2 extra days |
| `bulk_racking` | Bulk racking | 1 | $2,000 | $0 | +15% dry-goods storage |
| `smart_inventory` | Smart inventory | 3 | $40,000 | $50 | +20% all storage; auto-rotate oldest stock first |

### Production
| Key | Name | Era | Cost | Upkeep | Effect |
|---|---|---|---|---|---|
| `citrus_press` | Citrus press | 1 | $2,500 | $0 | 1 lemon makes 1.1 cases of lemonade (yield bonus) |
| `industrial_juicer` | Industrial juicer | 3 | $80,000 | $0 | Lemon yield +25% for all lemon recipes |
| `sugar_dissolver` | Syrup station | 2 | $10,000 | $0 | Sugar use −10% across recipes |
| `automation_line` | Automation line | 3 | $120,000 | $0 | Production upkeep −15% |
| `lean_ops` | Lean operations | 4 | $900,000 | $0 | Production upkeep −15% (stacks to −30%) |
| `oven` | Commercial oven | 2 | $12,000 | $15 | Unlocks bakery recipes |
| `carbonator` | Carbonation rig | 3 | $50,000 | $30 | Unlocks sparkling recipes (with Bottling Plant level or higher) |
| `zester` | Zester | 2 | $9,000 | $0 | Every lemon used yields 1 lemon peel (byproduct) |

### Market and brand
| Key | Name | Era | Cost | Upkeep | Effect |
|---|---|---|---|---|---|
| `roadside_sign` | Roadside sign | 1 | $800 | $0 | Presence +5% (Neighborhood) |
| `painted_stand` | Painted stand | 1 | $1,500 | $0 | +10 lemonade free depth |
| `billboard` | Billboard | 2 | $12,000 | $10 | Presence +10% (City) |
| `radio_spot` | Radio jingle | 3 | $90,000 | $80 | Presence +10% (Region); +5% lemonade depth everywhere |
| `tv_campaign` | TV campaign | 4 | $800,000 | $600 | Presence +10% (Nation) |
| `loyalty_app` | Loyalty app | 3 | $150,000 | $100 | Your lemonade sell depth recovers 10 points faster each night |
| `global_brand` | Global brand | 5 | $6M | $3,000 | Presence +10% everywhere; unlocks the "Global leader" victory |
| `mascot` | Mascot: Lemmy the Lemon | 2 | $20,000 | $0 | Holiday events +50% stronger for you |

### Intelligence
| Key | Name | Era | Cost | Upkeep | Effect |
|---|---|---|---|---|---|
| `weather_radio` | Weather radio | 1 | $1,000 | $0 | See tomorrow's weather event (if any) one day ahead |
| `farmers_almanac` | Farmer's almanac | 2 | $8,000 | $0 | Forecast 3 days; supply events too |
| `market_analyst` | Market analyst | 2 | $15,000 | $25 | 7-day moving average and trend on every price row |
| `rival_intel` | Rival intel | 3 | $50,000 | $40 | See rival telegraphs 2 days ahead, and their valuations exactly |
| `economist` | Chief economist | 4 | $500,000 | $300 | See the current economic cycle and its days left |

### Event resilience
| Key | Name | Era | Cost | Upkeep | Effect |
|---|---|---|---|---|---|
| `awnings` | Awnings and umbrellas | 1 | $1,500 | $0 | Rainy Week lemonade penalty halved (×0.75 → ×0.875) |
| `ice_machine` | Ice machine | 2 | $10,000 | $15 | Make up to 50 ice a day at $4 each; Heat Wave ice spike doesn't affect those |
| `generator` | Backup generator | 3 | $40,000 | $20 | Ignore "Power outage" |
| `orchard_lease` | Orchard lease | 3 | $100,000 | $100 | Lemon Blight effect halved; lemon input depth +20% |
| `insurance` | Business insurance | 3 | $80,000 | $150 | Any negative product-price event is floored at ×0.9 for you |
| `pr_team` | PR team | 4 | $400,000 | $250 | Rival price wars last half as long |

### Logistics and supply
| Key | Name | Era | Cost | Upkeep | Effect |
|---|---|---|---|---|---|
| `supplier_contract_1` | Supplier contract | 2 | $10,000 | $0 | All input asks −3% |
| `supplier_contract_2` | Bulk supplier network | 3 | $100,000 | $0 | Input depth +30%; asks −3% more |
| `supplier_contract_3` | Global sourcing | 5 | $5M | $0 | Input depth +60%; asks −4% more |
| `delivery_fleet` | Delivery fleet | 3 | $150,000 | $120 | Hub upkeep −25% |

### Finance
| Key | Name | Era | Cost | Upkeep | Effect |
|---|---|---|---|---|---|
| `bookkeeper` | Bookkeeper | 1 | $2,000 | $5 | Profit-and-loss breakdown in the day report |
| `accountant` | Accountant | 2 | $20,000 | $30 | Upkeep −5% (all) |
| `credit_line` | Credit line | 3 | $0 | interest | Borrow up to 20% of net worth at 0.5% a day (only if loans are approved: open question 29) |

### Automation and QoL
| Key | Name | Era | Cost | Upkeep | Effect |
|---|---|---|---|---|---|
| `order_book` | Order book | 1 | $500 | $0 | "Repeat yesterday's trades" button |
| `price_alerts` | Price alerts | 1 | $800 | $0 | Mark a row when a price crosses your threshold |
| `purchasing_mgr` | Purchasing manager | 2 | $25,000 | $60 | See section F |
| `sales_mgr` | Sales manager | 2 | $25,000 | $60 | See section F |
| `plant_mgr` | Plant manager | 3 | $80,000 | $150 | See section F |
| `logistics_mgr` | Logistics coordinator | 3 | $80,000 | $150 | See section F |
| `cfo` | CFO | 4 | $600,000 | $500 | See section F |
| `fast_forward` | Autopilot | 4 | $1M | $0 | Fast-forward up to 7 days (needs purchasing, sales and plant managers) |
| `retention_bonus` | Retention bonus | 4 | $300,000 | $200 | Managers cannot be poached |

---

## F. Managers

Each has one rule with player-set thresholds. They run in this order at the start of end-of-day: CFO reserve check, sales, purchasing, plant, logistics. All are deterministic.

| Manager | Rule the player sets | What it does |
|---|---|---|
| Sales manager | minimum bid (for example 95% of base) and "stay within free depth" on or off | sells finished goods down to the floor |
| Purchasing manager | maximum ask per input (for example 110% of base) | buys exactly the inputs needed for tomorrow's plan |
| Plant manager | skip recipes with a negative margin today (on or off) | sets the production plan order by today's margin |
| Logistics coordinator | buffer days (0 to 3) | keeps a stock buffer and rotates the freezer |
| CFO | cash reserve in days of upkeep | blocks manager spending below the reserve; the player's own actions are never blocked |

---

## G. Commodities (inputs)

| Key | Name | Category | Storage class | Base price | Shelf life | Era |
|---|---|---|---|---|---|---|
| `lemon` | Lemons | Citrus | cold | $20 | keeps (grandfathered) | 1 |
| `sugar` | Sugar | Sweetener | dry | $10 | keeps | 1 |
| `ice` | Ice | Base | frozen | $10 | melts nightly | 1 |
| `cup` | Cups | Packaging | dry | $10 | keeps | 1 |
| `lime` | Limes | Citrus | cold | $18 | 3 days | 1 |
| `mint` | Mint | Herb | cold | $12 | 3 days | 2 |
| `honey` | Honey | Sweetener | dry | $24 | keeps | 2 |
| `black_tea` | Black tea | Base | dry | $8 | keeps | 2 |
| `strawberry` | Strawberries | Berry | cold | $30 | 3 days | 2 |
| `flour` | Flour | Bakery | dry | $6 | keeps | 2 |
| `butter` | Butter | Bakery | cold | $14 | 5 days | 2 |
| `egg` | Eggs | Bakery | cold | $10 | 5 days | 2 |
| `watermelon` | Watermelon | Fruit | cold | $14 | 3 days | 3 |
| `ginger` | Ginger | Herb | cold | $16 | 5 days | 3 |
| `sparkling_water` | Sparkling water | Base | dry | $8 | keeps | 3 |
| `glass_bottle` | Glass bottles | Packaging | dry | $12 | keeps | 3 |
| `peach` | Peaches | Fruit | cold | $22 | 3 days | 4 |
| `agave` | Agave syrup | Sweetener | dry | $20 | keeps | 4 |
| `raspberry` | Raspberries | Berry | cold | $34 | 2 days | 4 |
| `coconut_water` | Coconut water | Base | dry | $18 | keeps | 5 |
| `lavender` | Lavender | Herb | dry | $45 | keeps | 5 |
| `lemon_peel` | Lemon peel | Byproduct | dry | n/a (not bought) | keeps | 2 (with Zester) |

Lemons keep today; making them perishable would change the base game, so they are grandfathered as "keeps" (open question 14).

---

## H. Products and recipes

Margin is at base prices: bid (90% of base, rounded down) minus the input asks (110% of base, rounded up), per case. Lemonade's $26 is the reference.

| Key | Product | Inputs (1 each unless noted) | Base | Margin | Depth vs lemonade | Needs | Era | Hook |
|---|---|---|---|---|---|---|---|---|
| `lemonade` | Lemonade | lemon, sugar, ice, cup | $90 | $26 | 100% | production | 1 | the classic |
| `limeade` | Limeade | lime, sugar, ice, cup | $88 | $26 | 35% | production | 1 | first unlock; same margin, new depth |
| `mint_lemonade` | Mint lemonade | lemon, sugar, mint, ice, cup | $110 | $30 | 30% | production | 2 | |
| `honey_lemonade` | Honey lemonade | lemon, honey, ice, cup | $115 | $32 | 30% | production | 2 | sugar-free: hedge against sugar spikes |
| `arnold_palmer` | Arnold Palmer | lemon, sugar, black tea, ice, cup | $100 | $26 | 40% | production | 2 | half tea, half lemonade |
| `strawberry_lemonade` | Strawberry lemonade | lemon, sugar, strawberry, ice, cup | $135 | $33 | 45% | production | 2 | the user's example; perishable input |
| `lemon_bars` | Lemon bars | lemon, sugar, flour, butter, egg | $105 | $27 | 40% | oven | 2 | **rain-proof**: not affected by Rainy Week |
| `candied_peel` | Candied peel | lemon peel, sugar | $35 | $20 | 10% | zester | 2 | byproduct: free peel from every lemon used |
| `frozen_lemonade` | Frozen lemonade | lemon, sugar, **3 ice**, cup | $125 | $35 | 30% | freezer | 3 | ice-hungry; Heat Wave ×1.6 |
| `watermelon_lemonade` | Watermelon lemonade | lemon, sugar, watermelon, ice, cup | $118 | $35 | 25% | production | 3 | summer only if seasons ship |
| `ginger_lemonade` | Ginger lemonade | lemon, honey, ginger, ice, cup | $128 | $26 | 15% | production | 3 | "Health trend" event ×1.4 |
| `sparkling_lemonade` | Sparkling lemonade (bottled) | lemon, sugar, sparkling water, glass bottle | $108 | $41 | 50% | carbonation rig, Bottling Plant level or higher | 3 | no ice; shelf-stable; the scale product |
| `peach_iced_tea` | Peach iced tea | lemon, peach, black tea, sugar, ice, cup | $125 | $23 | 35% | production | 4 | many inputs; lemon-light |
| `raspberry_fizz` | Raspberry fizz | lemon, agave, raspberry, sparkling water, glass bottle | $170 | $48 | 15% | carbonation rig | 4 | premium, fragile supply (2-day raspberries) |
| `hot_honey_lemon` | Hot honey lemon | lemon, honey, black tea, cup | $95 | $16 | 25% | production | 4 | **counter-cyclical**: Rainy Week ×1.5, Cold Snap ×1.6 |
| `coconut_lime_cooler` | Coconut lime cooler | lime, coconut water, agave, ice, glass bottle | $140 | $39 | 20% | carbonation rig | 5 | lemon-free: immune to Lemon Blight |
| `lavender_lemonade` | Lavender lemonade | lemon, honey, lavender, ice, glass bottle | $230 | $83 | 5% | production | 5 | luxury; tiny market, huge margin |

---

## I. Market events (new; the existing six stay)

All effects are price multipliers or depth multipliers; none destroy inventory. Conflicts are derived automatically (P0 rule: opposite directions on the same commodity never overlap).

| Key | Name | Duration | Effect | Era |
|---|---|---|---|---|
| `strawberry_season` | Strawberry season | 5 | strawberry ×0.6 | 2 |
| `mint_frost` | Mint frost | 3 | mint ×1.8 | 2 |
| `bee_decline` | Bee decline | 4 | honey ×1.6 | 2 |
| `street_fair` | Street fair | 1 | all drinks ×1.25, lemonade depth +50% | 1 |
| `influencer` | Influencer post | 2 | one random product ×1.5 | 2 |
| `health_trend` | Health trend | 7 | sugar drinks ×0.9, honey and ginger drinks ×1.3 | 3 |
| `cold_snap` | Cold snap | 3 | cold drinks ×0.75, hot honey lemon ×1.6 | 3 |
| `heat_dome` | Heat dome | 4 | all cold drinks ×1.4, ice ×1.6, frozen lemonade ×1.6 | 3 |
| `tea_tariff` | Tea tariff | 5 | black tea ×1.5 | 3 |
| `glass_shortage` | Glass shortage | 3 | glass bottle ×1.7 | 3 |
| `sports_final` | Big game day | 1 | all drinks depth +100% | 2 |
| `power_outage` | Power outage | 1 | production −50% (the generator negates this) | 3 |
| `port_strike` | Port strike | 4 | imported inputs (coconut water, agave, lavender) ×1.5, input depth −30% | 5 |
| `citrus_canker` | Citrus canker | 4 | lemon ×1.5, lime ×1.5 | 4 |
| `bumper_crop` | Bumper crop | 4 | lemon ×0.7, lime ×0.8 | 2 |
| `food_blog` | Food blog feature | 3 | lemon bars and candied peel ×1.5 | 2 |
| `sugar_tax` | Sugar tax | 10 | sugar drinks ×0.9 (Nation only) | 4 |
| `world_expo` | World expo | 3 | all products ×1.3, depth +30% | 5 |

**Economic cycles** (long regimes, at most one active, a 3% chance per day to start when none is active):

| Key | Name | Duration | Effect |
|---|---|---|---|
| `boom` | Boom | 15 to 25 days | all product depth +15%, rival valuations +20% (buyouts cost more) |
| `recession` | Recession | 15 to 25 days | all product depth −15%, rival valuations −25% (buyout bargains) |
| `inflation` | Inflation | 20 days | all prices drift up 1% a day, upkeep +10% |
| `stable` | Stable | default | no effect |

---

## J. Contracts (example rows; if contracts ship)

An offer lasts 3 days. Accepting locks in delivery by a deadline for a price above base. Failing costs a penalty of 20% of the contract value, never more than 5% of net worth.

| Key | Client | Order | Deadline | Price | Era |
|---|---|---|---|---|---|
| `block_party` | Block party | 60 lemonade | 3 days | $105/case | 1 |
| `little_league` | Little League | 100 limeade | 5 days | $100 | 1 |
| `office_park` | Office park | 400 Arnold Palmer | 7 days | $115 | 2 |
| `farmers_market` | Farmers' market | 200 lemon bars | 5 days | $120 | 2 |
| `stadium` | Stadium | 2,000 lemonade | 10 days | $105 | 3 |
| `airline` | Airline | 5,000 sparkling lemonade | 14 days | $125 | 4 |
| `hotel_chain` | Hotel chain | 1,500 lavender lemonade | 14 days | $260 | 5 |
| `world_cup` | Global games | 50,000 mixed drinks | 21 days | +20% over base | 5 |

---

## K. Achievements (about 70)

Tiers: **B** bronze, **S** silver, **G** gold. Hidden ones show as "???" until unlocked. Every check is a typed predicate over the game, stats or run history. Those marked ★ need no new systems, so they can ship in phase 1.

### Wealth
| Key | Name | Tier | Condition |
|---|---|---|---|
| `nw_5k` ★ | Pocket money | B | net worth $5,000 |
| `nw_10k` ★ | Five figures | B | $10,000 |
| `nw_25k` ★ | Comfortable | B | $25,000 |
| `nw_100k` ★ | Six figures | S | $100,000 |
| `nw_1m` | Lemonade millionaire | S | $1,000,000 |
| `nw_10m` | Big squeeze | G | $10,000,000 |
| `nw_100m` | Citrus tycoon | G | $100,000,000 |
| `nw_1b` (hidden) | Lemon billionaire | G | $1,000,000,000 |
| `cash_pile` ★ | Rainy-day fund | B | hold $10,000 in cash with no stock |
| `peak_day` ★ | Record day | S | earn $5,000 profit in one day |

### Survival and runs
| Key | Name | Tier | Condition |
|---|---|---|---|
| `day_7` ★ | First week | B | reach day 7 |
| `day_30` ★ | Month one | B | reach day 30 |
| `day_60` ★ | Seasoned | S | reach day 60 |
| `day_100` ★ | Centurion | S | reach day 100 |
| `day_200` | Institution | G | reach day 200 |
| `day_365` (hidden) | A year of lemonade | G | reach day 365 |
| `runs_5` ★ | Serial entrepreneur | B | finish 5 runs |
| `runs_25` ★ | Can't stop squeezing | S | finish 25 runs |
| `new_best` ★ | Personal best | B | beat your best score |
| `top_10` ★ | Hall of fame | G | a top-10 run on the global board |

### Production and capacity
| Key | Name | Tier | Condition |
|---|---|---|---|
| `made_1k` ★ | First thousand | B | produce 1,000 lemonade in a run |
| `made_10k` ★ | Ten thousand cups | S | produce 10,000 |
| `made_100k` | Assembly line | G | produce 100,000 |
| `made_1m` | Lemonade river | G | produce 1,000,000 |
| `big_batch` ★ | Big batch | S | produce 500 in one day |
| `full_house` ★ | Full house | B | every warehouse at 100% at once |
| `stocked_1k` ★ | Stockpile | S | hold 1,000 cases in total |
| `stocked_10k` | Hoarder | G | hold 10,000 cases in total |
| `no_idle` ★ | Humming | S | production at full capacity 7 days running |

### Facilities and upgrades
| Key | Name | Tier | Condition |
|---|---|---|---|
| `first_expand` ★ | Growing | B | build a building |
| `first_upgrade` ★ | Moving up | B | upgrade a facility |
| `max_production` ★ | Factory floor | S | production at max level and max buildings (per current era cap) |
| `max_warehouse` ★ | Room to spare | S | warehouses at max level and max buildings |
| `all_maxed` ★ | Everything maxed | G | both of the above |
| `buildings_50` | Campus | S | own 50 buildings |
| `buildings_100` | Industrial park | G | own 100 buildings |
| `first_upgrade_item` | Gadget | B | buy an upgrade |
| `freshness_set` | Ice cold | S | own all freshness upgrades |
| `upgrades_all` | Fully loaded | G | own every upgrade |
| `sold_building` ★ | Downsizing | B | sell a building |

### Trading and market
| Key | Name | Tier | Condition |
|---|---|---|---|
| `buy_low` ★ | Bargain hunter | B | buy an input at 70% of base or lower |
| `sell_high` ★ | Top of the market | S | sell lemonade at 150% of base or higher |
| `heat_seller` ★ | Hot seller | B | sell 100 lemonade during a Heat Wave |
| `rain_profit` ★ | Singing in the rain | S | end a Rainy Week day with a profit |
| `every_event` ★ | Seen it all | S | experience every event in the table |
| `no_slippage` ★ | Light touch | S | sell 500 lemonade in a run without paying price impact |
| `volume_trader` ★ | Market mover | B | pay price impact for the first time (tooltip explains depth) |
| `avg_cost_win` ★ | Bought right | S | sell lemonade at twice its average cost |

### Rivals and territories
| Key | Name | Tier | Condition |
|---|---|---|---|
| `first_buyout` | Acquisition | B | buy out a rival |
| `lucy_bought` | Lemonade from a kid | B | buy Lil' Lucy's stand |
| `own_neighborhood` | Block boss | S | 100% of the Neighborhood |
| `enter_city` | Big city | B | enter Citrus City |
| `own_city` | Mayor of lemonade | S | 50% of the City |
| `enter_region` | Regional | S | enter the Region |
| `enter_nation` | National brand | S | enter the Nation |
| `enter_world` | Going global | G | enter the World |
| `hostile` | Hostile takeover | S | complete a hostile buyout |
| `price_war_won` | Price warrior | S | keep your share through a price war |
| `victory` | Global leader | G | win the game |
| `underdog` (hidden) | Underdog | G | win having never bought Global Citrus Holdings |

### Recipes and products
| Key | Name | Tier | Condition |
|---|---|---|---|
| `first_recipe` | New flavor | B | produce a second product |
| `five_products` | Menu board | S | sell five products in one day |
| `all_recipes` | Master mixer | G | produce every recipe in a run |
| `strawberry_1k` | Berry good | S | sell 1,000 strawberry lemonade |
| `byproduct` | Nothing wasted | B | sell candied peel |
| `hedge` | Hedged | S | profit on a Rainy Week day from rain-proof products |
| `luxury` | Luxury line | S | sell 100 lavender lemonade |

### Oddities (mostly hidden)
| Key | Name | Tier | Condition |
|---|---|---|---|
| `just_ice` ★ (hidden) | Just ice | B | go bankrupt holding only ice |
| `close_call` ★ | Close call | S | survive an end of day with $1 to $9 left |
| `comeback` ★ | Lemons into lemonade | G | recover from under $100 to $10,000 in one run |
| `speedrun` ★ | Speedrun | S | net worth $10,000 by day 15 |
| `patience` ★ (hidden) | Patience | B | end 5 days in a row without trading |
| `give_up_rich` ★ (hidden) | Quit while ahead | B | give up with over $50,000 |
| `day_one_loss` ★ (hidden) | Tough crowd | B | go bankrupt on day 1 to 3 |
| `sugar_rush` ★ | Sugar rush | B | hold 500 sugar |

---

## L. Name bank (for future rows)

**Rivals:** Tangy Tom's, Rind & Co., Pith Perfect, The Juice Loose, Squeezy Does It, Citrus Circus, Pucker Palace, Zesty Business, Lemon Aid Society, Sip City, The Sour Patch Co-op, Peel Good Inc., Yellow Submarine Drinks, Bitter Rivals Ltd.

**Upgrades:** Solar panels (upkeep −5%), Neon sign, Espresso machine (joke: unlocks nothing, +1 achievement), Drive-thru window (depth +5%), Recycling program (cups −5%), Loyalty punch cards, Franchise manual (prestige, later).
