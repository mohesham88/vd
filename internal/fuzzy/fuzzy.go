package fuzzy

import (
	"github.com/sahilm/fuzzy"
)

func Search(pattern string, entries []string) fuzzy.Matches {
	return fuzzy.Find(pattern, entries)
}
