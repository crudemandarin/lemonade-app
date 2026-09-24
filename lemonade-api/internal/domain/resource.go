package domain

// Resource is one of the five tradable goods. Each has its own warehouse.
type Resource string

const (
	Lemon    Resource = "lemon"
	Sugar    Resource = "sugar"
	Ice      Resource = "ice"
	Cup      Resource = "cup"
	Lemonade Resource = "lemonade"
)

// Resources lists all resources in display order.
var Resources = []Resource{Lemon, Sugar, Ice, Cup, Lemonade}

// Inputs are the raw resources consumed by production, in recipe order.
var Inputs = []Resource{Lemon, Sugar, Ice, Cup}

func (r Resource) Valid() bool {
	switch r {
	case Lemon, Sugar, Ice, Cup, Lemonade:
		return true
	default:
		return false
	}
}

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
)
