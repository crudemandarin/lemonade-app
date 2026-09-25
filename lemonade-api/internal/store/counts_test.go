package store

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Rows saved before the commodity catalog store timeline stock and price log prices
// as 5-element arrays in the order lemon, sugar, ice, cup, lemonade.
func TestCountsReadTheLegacyArrayShape(t *testing.T) {
	var p pointRow
	if err := json.Unmarshal([]byte(`{"d":3,"k":"end_day","c":900,"s":[4,0,2,0,7]}`), &p); err != nil {
		t.Fatal(err)
	}
	want := counts{"lemon": 4, "ice": 2, "lemonade": 7}
	if !reflect.DeepEqual(p.Stock, want) {
		t.Fatalf("stock = %v, want %v", p.Stock, want)
	}

	var pr priceRow
	if err := json.Unmarshal([]byte(`{"day":2,"prices":[21,9,10,11,95],"events":["holiday"]}`), &pr); err != nil {
		t.Fatal(err)
	}
	if got := fromCounts(pr.Prices); got["lemon"] != 21 || got["cup"] != 11 || got["lemonade"] != 95 {
		t.Fatalf("prices = %v", got)
	}
}

func TestCountsRoundTripAsAnObjectWithoutZeros(t *testing.T) {
	b, err := json.Marshal(counts{"lemon": 3, "sugar": 0})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"lemon":3}` {
		t.Fatalf("marshalled %s", b)
	}
	var back counts
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, counts{"lemon": 3}) {
		t.Fatalf("round trip = %v", back)
	}
}
