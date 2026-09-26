package content

// Ingredient is one input of a recipe: Qty cases of the commodity Key per batch.
type Ingredient struct {
	Key string
	Qty int
}

// RecipeDef turns inputs into OutputQty cases of Output per batch. Inputs are a
// slice, not a map, so production always takes them in the same order.
type RecipeDef struct {
	Key       string
	Name      string
	Output    string
	OutputQty int
	Inputs    []Ingredient
	// Learning a recipe is an era-gated purchase: it is offered from Era on, for LearnCost.
	// Unlock names a feature an upgrade must have unlocked (for example "bakery", from the
	// oven); MinProductionLevel is the production tier it needs. The lemonade recipe is
	// known from the start (Era 0).
	Era                int
	LearnCost          int
	Unlock             string
	MinProductionLevel int
	// Text is a line about it for the Production page.
	Text string
}

// Recipes is the recipe table. Lemonade is known from the start; the rest are learned.
var Recipes = []RecipeDef{
	{
		Key: "lemonade", Name: "Lemonade", Output: "lemonade", OutputQty: 1,
		Inputs: []Ingredient{{Key: "lemon", Qty: 1}, {Key: "sugar", Qty: 1}, {Key: "ice", Qty: 1}, {Key: "cup", Qty: 1}},
		Text:   "The classic.",
	},
	{
		Key: "limeade", Name: "Limeade", Output: "limeade", OutputQty: 1, Era: 1, LearnCost: 800,
		Inputs: []Ingredient{{Key: "lime", Qty: 1}, {Key: "sugar", Qty: 1}, {Key: "ice", Qty: 1}, {Key: "cup", Qty: 1}},
		Text:   "The same margin as lemonade in a market of its own.",
	},
	{
		Key: "mint_lemonade", Name: "Mint lemonade", Output: "mint_lemonade", OutputQty: 1, Era: 2, LearnCost: 4000,
		Inputs: []Ingredient{{Key: "lemon", Qty: 1}, {Key: "sugar", Qty: 1}, {Key: "mint", Qty: 1}, {Key: "ice", Qty: 1}, {Key: "cup", Qty: 1}},
		Text:   "Fresh mint spoils in 3 days.",
	},
	{
		Key: "honey_lemonade", Name: "Honey lemonade", Output: "honey_lemonade", OutputQty: 1, Era: 2, LearnCost: 4000,
		Inputs: []Ingredient{{Key: "lemon", Qty: 1}, {Key: "honey", Qty: 1}, {Key: "ice", Qty: 1}, {Key: "cup", Qty: 1}},
		Text:   "No sugar: a hedge against a sugar spike.",
	},
	{
		Key: "arnold_palmer", Name: "Arnold Palmer", Output: "arnold_palmer", OutputQty: 1, Era: 2, LearnCost: 4000,
		Inputs: []Ingredient{{Key: "lemon", Qty: 1}, {Key: "sugar", Qty: 1}, {Key: "black_tea", Qty: 1}, {Key: "ice", Qty: 1}, {Key: "cup", Qty: 1}},
		Text:   "Half tea, half lemonade.",
	},
	{
		Key: "strawberry_lemonade", Name: "Strawberry lemonade", Output: "strawberry_lemonade", OutputQty: 1, Era: 2, LearnCost: 6000,
		Inputs: []Ingredient{{Key: "lemon", Qty: 1}, {Key: "sugar", Qty: 1}, {Key: "strawberry", Qty: 1}, {Key: "ice", Qty: 1}, {Key: "cup", Qty: 1}},
		Text:   "The best margin of the launch drinks; the strawberries spoil in 3 days.",
	},
	{
		Key: "lemon_bars", Name: "Lemon bars", Output: "lemon_bars", OutputQty: 1, Era: 2, LearnCost: 5000, Unlock: "bakery",
		Inputs: []Ingredient{{Key: "lemon", Qty: 1}, {Key: "sugar", Qty: 1}, {Key: "flour", Qty: 1}, {Key: "butter", Qty: 1}, {Key: "egg", Qty: 1}},
		Text:   "Needs the oven. Rain does not hurt them.",
	},
	{
		Key: "candied_peel", Name: "Candied peel", Output: "candied_peel", OutputQty: 1, Era: 2, LearnCost: 2000, Unlock: "byproduct_lemon_peel",
		Inputs: []Ingredient{{Key: "lemon_peel", Qty: 1}, {Key: "sugar", Qty: 1}},
		Text:   "Made from the peel the zester saves.",
	},
}
