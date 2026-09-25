package store

import (
	"encoding/json"

	"lemonade-api/internal/domain"
	"lemonade-api/internal/domain/content"
)

// counts is a per-commodity number (timeline stock, a day's prices) as stored in
// JSONB: an object keyed by commodity, with zeros left out to keep long timelines
// small. Rows saved before the commodity catalog hold a 5-element array in the
// legacy order (lemon, sugar, ice, cup, lemonade); UnmarshalJSON reads both.
type counts map[string]int

func (c counts) MarshalJSON() ([]byte, error) {
	out := make(map[string]int, len(c))
	for k, v := range c {
		if v != 0 {
			out[k] = v
		}
	}
	return json.Marshal(out)
}

func (c *counts) UnmarshalJSON(b []byte) error {
	var arr []int
	if err := json.Unmarshal(b, &arr); err == nil {
		m := make(counts, len(arr))
		for i, v := range arr {
			if i < len(content.LegacyOrder) && v != 0 {
				m[content.LegacyOrder[i]] = v
			}
		}
		*c = m
		return nil
	}
	var m map[string]int
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	for k, v := range m {
		if v == 0 {
			delete(m, k)
		}
	}
	*c = counts(m)
	return nil
}

func toCounts(m map[domain.Resource]int) counts {
	out := make(counts, len(m))
	for r, v := range m {
		out[string(r)] = v
	}
	return out
}

func fromCounts(c counts) map[domain.Resource]int {
	out := make(map[domain.Resource]int, len(c))
	for k, v := range c {
		out[domain.Resource(k)] = v
	}
	return out
}
