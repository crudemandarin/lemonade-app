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
}

// Recipes is the recipe table. Lemonade is the only recipe until late game
// Products B.
var Recipes = []RecipeDef{
	{
		Key: "lemonade", Name: "Lemonade", Output: "lemonade", OutputQty: 1,
		Inputs: []Ingredient{{Key: "lemon", Qty: 1}, {Key: "sugar", Qty: 1}, {Key: "ice", Qty: 1}, {Key: "cup", Qty: 1}},
	},
}
