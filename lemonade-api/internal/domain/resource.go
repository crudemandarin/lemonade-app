package domain

// Resource is a commodity key from the content catalog. The constants name the
// original five, which rules and events still refer to directly.
type Resource string

const (
	Lemon    Resource = "lemon"
	Sugar    Resource = "sugar"
	Ice      Resource = "ice"
	Cup      Resource = "cup"
	Lemonade Resource = "lemonade"
)

// The list of commodities, their order and the recipe are catalog data: see
// Config.Resources, Config.Valid and Config.Inputs (catalog.go).

// FacilityType is one of the two upgradeable facility groups.
type FacilityType string

const (
	Warehouse  FacilityType = "warehouse"
	Production FacilityType = "production"
)

func (k FacilityType) Valid() bool {
	return k == Warehouse || k == Production
}

// Status is the game's lifecycle state.
type Status string

const (
	StatusActive   Status = "active"
	StatusBankrupt Status = "bankrupt"
	StatusGaveUp   Status = "gave_up"
)
