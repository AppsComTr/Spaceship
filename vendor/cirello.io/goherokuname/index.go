package goherokuname

var (
	adjectivesIdx map[byte][]int
	nounsIdx      map[byte][]int
)

func init() {
	adjectivesIdx = extractIndexes(adjectives)
	nounsIdx = extractIndexes(nouns)
}

func extractIndexes(dict []string) map[byte][]int {
	idx := map[byte][]int{
		'a': {0},
	}

	lastLetter := byte('a')
	var i int
	var w string
	for i, w = range dict {
		if w[0] != lastLetter {
			idx[lastLetter] = append(idx[lastLetter], i-1)
			idx[w[0]] = []int{i}
			lastLetter = w[0]
		}
	}
	idx[lastLetter] = append(idx[lastLetter], i)
	return idx
}
