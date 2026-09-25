package content

import "testing"

func TestAchievementTableIsWellFormed(t *testing.T) {
	categories := map[string]bool{}
	for _, c := range AchievementCategories {
		if c.Key == "" || c.Name == "" || categories[c.Key] {
			t.Errorf("bad or duplicate category %+v", c)
		}
		categories[c.Key] = true
	}
	tiers := map[string]bool{TierBronze: true, TierSilver: true, TierGold: true}
	seen := map[string]bool{}
	for _, a := range Achievements {
		if a.Key == "" || a.Name == "" || a.Description == "" {
			t.Errorf("%+v needs a key, a name and a description", a)
		}
		if seen[a.Key] {
			t.Errorf("duplicate achievement key %q", a.Key)
		}
		seen[a.Key] = true
		if !categories[a.Category] {
			t.Errorf("%s: unknown category %q", a.Key, a.Category)
		}
		if !tiers[a.Tier] {
			t.Errorf("%s: unknown tier %q", a.Key, a.Tier)
		}
		if a.Check.Kind == "" {
			t.Errorf("%s: no check", a.Key)
		}
	}
}
