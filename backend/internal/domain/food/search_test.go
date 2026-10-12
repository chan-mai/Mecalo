package food

import (
	"slices"
	"testing"
)

func TestBigrams(t *testing.T) {
	got := Bigrams(SearchKey("いきなり！ステーキ"))
	want := []string{"いき", "きな", "なり", "りす", "すて", "てー", "ーき"}
	if !slices.Equal(got, want) {
		t.Errorf("Bigrams = %v, want %v", got, want)
	}
	if !slices.Equal(Bigrams(SearchKey("いきなりステーキ")), want) {
		t.Error("symbol should not change bigrams")
	}
	if len(Bigrams("a")) != 0 || len(Bigrams("")) != 0 {
		t.Error("single rune has no bigram")
	}
	if got := Bigrams("ああああ"); !slices.Equal(got, []string{"ああ"}) {
		t.Errorf("dedupe = %v", got)
	}
}

func TestSimilarity(t *testing.T) {
	c, d := Similarity(3, 4, 6)
	if c != 0.75 || d != 0.6 {
		t.Errorf("Similarity = %v, %v", c, d)
	}
	if c, d := Similarity(0, 0, 3); c != 0 || d != 0 {
		t.Error("empty query")
	}
}
