package destiny

import (
	"github.com/eliwjones/thebox/util/structs"

	"testing"
)

func dummyEdges() []structs.Maximum {
	edges := []structs.Maximum{}
	edges = append(edges, structs.Maximum{Underlying: "GOOG", MaximumBid: 1100, OptionAsk: 500})
	edges = append(edges, structs.Maximum{Underlying: "GOOG", MaximumBid: 500, OptionAsk: 500})
	edges = append(edges, structs.Maximum{Underlying: "AAPL", MaximumBid: 1000, OptionAsk: 500})
	edges = append(edges, structs.Maximum{Underlying: "BABA", MaximumBid: 1000, OptionAsk: 500})

	return edges
}

func Test_Destiny_filterEdgesByMultiplier(t *testing.T) {
	multiplier := float64(2)
	edges := filterEdgesByMultiplier(dummyEdges(), multiplier)

	if len(edges) == len(dummyEdges()) {
		t.Errorf("Expected some edges to be filtered out.")
	}

	for _, edge := range edges {
		em := float64(edge.MaximumBid) / float64(edge.OptionAsk)
		if em < multiplier {
			t.Errorf("Expecting MaximumBid/OptionAsk < %.4f, Got: %.4f", multiplier, em)
		}
	}
}

func Test_Destiny_filterEdgesByUnderlying(t *testing.T) {
	underlying := "GOOG"
	edges := filterEdgesByUnderlying(dummyEdges(), underlying)

	if len(edges) == len(dummyEdges()) {
		t.Errorf("Expected some edges to be filtered out.")
	}

	for _, edge := range edges {
		if edge.Underlying != underlying {
			t.Errorf("Expected: %s, Got: %s", underlying, edge.Underlying)
		}
	}
}
