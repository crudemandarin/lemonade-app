package domain

import (
	"errors"
	"testing"
)

func TestBuyClamped(t *testing.T) {
	// Lemon ask is $22 at the start; a Pantry holds 10.
	tests := []struct {
		name    string
		capital int
		stock   int
		qty     int
		want    int   // cases bought
		err     error // expected error when want is 0
	}{
		{"enough of everything", 1000, 0, 5, 5, nil},
		{"limited by cash", 100, 0, 50, 4, nil},
		{"limited by space", 1000, 6, 50, 4, nil},
		{"both: cash binds", 44, 5, 50, 2, nil},
		{"both: space binds", 1000, 8, 50, 2, nil},
		{"no cash", 21, 0, 5, 0, ErrInsufficientFunds},
		{"no space", 1000, 10, 5, 0, ErrCapacityExceeded},
		{"no cash and no space reports cash", 0, 10, 5, 0, ErrInsufficientFunds},
		{"all", 1000, 0, 1_000_000, 10, nil},
	}
	for _, tt := range tests {
		g, cfg := newTestGame()
		g.Capital, g.Inventory[Lemon] = tt.capital, tt.stock
		err := BuyClamped(&g, cfg, Lemon, tt.qty)
		if tt.want == 0 {
			if !errors.Is(err, tt.err) || g.Capital != tt.capital || g.Inventory[Lemon] != tt.stock {
				t.Errorf("%s: err=%v capital=%d stock=%d, want %v and no change", tt.name, err, g.Capital, g.Inventory[Lemon], tt.err)
			}
			continue
		}
		if err != nil || g.Inventory[Lemon] != tt.stock+tt.want || g.Capital != tt.capital-22*tt.want {
			t.Errorf("%s: err=%v stock=%d capital=%d, want +%d cases", tt.name, err, g.Inventory[Lemon], g.Capital, tt.want)
		}
	}
}

func TestSellClamped(t *testing.T) {
	g, cfg := newTestGame()
	g.Inventory[Lemon] = 4
	if err := SellClamped(&g, cfg, Lemon, 50); err != nil {
		t.Fatal(err)
	}
	if g.Inventory[Lemon] != 0 || g.Capital != 1000+4*18 {
		t.Fatalf("stock=%d capital=%d, want everything sold at the $18 bid", g.Inventory[Lemon], g.Capital)
	}
	if err := SellClamped(&g, cfg, Lemon, 5); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("selling from empty: %v, want ErrInsufficientStock", err)
	}
	g.Inventory[Lemon] = 9
	if err := SellClamped(&g, cfg, Lemon, 3); err != nil || g.Inventory[Lemon] != 6 {
		t.Fatalf("partial: err=%v stock=%d", err, g.Inventory[Lemon])
	}
}

func TestClampedTradesKeepTheOriginalValidation(t *testing.T) {
	g, cfg := newTestGame()
	if err := BuyClamped(&g, cfg, Lemon, 0); !errors.Is(err, ErrInvalidQuantity) {
		t.Errorf("buy 0: %v", err)
	}
	if err := SellClamped(&g, cfg, Lemon, -1); !errors.Is(err, ErrInvalidQuantity) {
		t.Errorf("sell -1: %v", err)
	}
	g.Status = StatusBankrupt
	if err := BuyClamped(&g, cfg, Lemon, 1); !errors.Is(err, ErrGameOver) {
		t.Errorf("buy when over: %v", err)
	}
	// Without clamp the plain trade still fails all-or-nothing.
	g, cfg = newTestGame()
	if err := Buy(&g, cfg, Lemon, 50); !errors.Is(err, ErrInsufficientFunds) && !errors.Is(err, ErrCapacityExceeded) {
		t.Errorf("unclamped oversize buy: %v", err)
	}
}
