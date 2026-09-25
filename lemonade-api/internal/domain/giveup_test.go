package domain

import (
	"errors"
	"testing"
)

func TestGiveUpEndsTheRun(t *testing.T) {
	g, cfg := newTestGame()
	Buy(&g, cfg, Lemon, 3)
	day, capital := g.Day, g.Capital

	if err := GiveUp(&g); err != nil {
		t.Fatal(err)
	}
	if g.Status != StatusGaveUp || g.Day != day || g.Capital != capital {
		t.Fatalf("status=%s day=%d capital=%d", g.Status, g.Day, g.Capital)
	}
	// Nothing is sold or charged: the score reflects the state the player left.
	if g.Inventory[Lemon] != 3 {
		t.Fatal("giving up changed the stock")
	}
}

func TestGiveUpTwiceIsRefused(t *testing.T) {
	g, _ := newTestGame()
	if err := GiveUp(&g); err != nil {
		t.Fatal(err)
	}
	if err := GiveUp(&g); !errors.Is(err, ErrGameOver) {
		t.Fatalf("second give up: %v, want ErrGameOver", err)
	}
	g.Status = StatusBankrupt
	if err := GiveUp(&g); !errors.Is(err, ErrGameOver) {
		t.Fatalf("give up after bankruptcy: %v", err)
	}
}

// Every action refuses to run unless the game is active, whichever way it ended.
func TestNothingWorksAfterTheGameEnds(t *testing.T) {
	for _, status := range []Status{StatusBankrupt, StatusGaveUp} {
		g, cfg := newTestGame()
		g.Inventory[Lemon] = 2
		g.Status = status
		calls := map[string]error{
			"buy":           Buy(&g, cfg, Lemon, 1),
			"sell":          Sell(&g, cfg, Lemon, 1),
			"buy clamped":   BuyClamped(&g, cfg, Lemon, 1),
			"sell clamped":  SellClamped(&g, cfg, Lemon, 1),
			"expand":        Expand(&g, cfg, Production, ""),
			"upgrade":       Upgrade(&g, cfg, Production),
			"sell building": SellFacility(&g, cfg, Production, ""),
			"give up":       GiveUp(&g),
		}
		_, calls["end day"] = EndDay(&g, cfg)
		for name, err := range calls {
			if !errors.Is(err, ErrGameOver) {
				t.Errorf("%s while %s: %v, want ErrGameOver", name, status, err)
			}
		}
	}
}
