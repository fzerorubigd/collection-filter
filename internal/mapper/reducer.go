package mapper

import (
	"fmt"
	"sort"

	"github.com/fzerorubigd/gobgg"
)

type Reducer = func(thing *gobgg.ThingResult) bool

type Less = func(thing1, thing2 *gobgg.ThingResult) bool

func WeightReducer(min, max int) Reducer {
	minF := float64(min) / 10
	maxF := float64(max) / 10
	return func(thing *gobgg.ThingResult) bool {
		if thing.AverageWeight < minF {
			return false
		}
		if thing.AverageWeight > maxF {
			return false
		}

		return true
	}
}

// It means it is not "not recommended"
func RecommendedFor(pl int) Reducer {
	return func(thing *gobgg.ThingResult) bool {
		for _, sp := range thing.SuggestedPlayerCount {
			if sp.NumPlayers == fmt.Sprint(pl) {
				su, _, _ := sp.Suggestion()
				switch su {
				case "Not Recommended":
					return false
				default:
					return true
				}
			}
		}
		return false
	}
}

func OnlyBoardGame() Reducer {
	return func(thing *gobgg.ThingResult) bool {
		return thing.Type == gobgg.BoardGameType
	}
}

func Reduce(all []gobgg.ThingResult, reducers ...Reducer) []gobgg.ThingResult {
	result := make([]gobgg.ThingResult, 0, len(all))
bigLoop:
	for i := range all {
		for red := range reducers {
			if !reducers[red](&all[i]) {
				continue bigLoop
			}
		}

		result = append(result, all[i])
	}

	return result

}

func suggestedPlayerCount(sp []gobgg.SuggestedPlayerCount, pl int) (string, int, float32) {
	for i := range sp {
		if sp[i].NumPlayers == fmt.Sprint(pl) {
			return sp[i].Suggestion()
		}
	}

	return "Not Recommended", 0, 0
}

var compareStr = map[string]int{
	"Best":            2,
	"Recommended":     1,
	"Not Recommended": 0,
}

func ComparatorPlayerCount(pl int) Less {
	return func(thing1, thing2 *gobgg.ThingResult) bool {
		t1S, t1N, t1P := suggestedPlayerCount(thing1.SuggestedPlayerCount, pl)
		t2S, t2N, t2P := suggestedPlayerCount(thing2.SuggestedPlayerCount, pl)

		if compareStr[t1S] != compareStr[t2S] {
			return compareStr[t1S] < compareStr[t2S]
		}

		if t1N != t2N {
			return t1N < t2N
		}

		return t1P < t2P
	}
}

func Sort(all []gobgg.ThingResult, comparator Less) {
	sort.Slice(all, func(i, j int) bool {
		return !comparator(&all[i], &all[j])
	})
}
