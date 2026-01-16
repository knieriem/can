package filter

import (
	"slices"
	"sort"

	"github.com/knieriem/can"
)

// Interval represents an inclusive range of CAN IDs.
type Interval struct {
	Start, End uint32
}

// Filter represents a set of non-overlapping, sorted CAN ID ranges.
type Filter []Interval

// NewFilter processes a list of [can.MsgFilter]s into a minimized set of allowed ranges.
// It assumes an "OR" logic for inclusions and "AND NOT" logic for inverted filters.
func (f *Filter) Add(input []can.MsgFilter, extFrame bool) {

	maxID := uint32(0x7FF)
	if extFrame {
		maxID = 0x1FFF_FFFF
	}

	for i := range input {
		mf := &input[i]
		if extFrame != mf.ExtFrame {
			continue
		}
		from, to, ok := mf.Range()
		if !ok {
			continue
		}
		if mf.Invert {
			if len(*f) == 0 {
				// If the first rule is an exclusion,
				// we assume a starting state of "allow all"
				f.insert(0, maxID)
			}
			f.cut(from, to)
		} else {
			f.insert(from, to)
		}
	}
}

// insert adds a range and merges overlapping or adjacent intervals.
func (f *Filter) insert(start, end uint32) {
	idx := sort.Search(len(*f), func(i int) bool {
		return (*f)[i].End >= start
	})

	actualStart, actualEnd := start, end
	mergeEnd := idx

	list := *f
	for i := idx; i < len(*f); i++ {
		// +1 allows merging adjacent ranges like [100, 100] and [101, 101]
		if list[i].Start > end+1 {
			break
		}
		if list[i].Start < actualStart {
			actualStart = list[i].Start
		}
		if list[i].End > actualEnd {
			actualEnd = list[i].End
		}
		mergeEnd = i + 1
	}

	*f = slices.Replace(list, idx, mergeEnd, Interval{actualStart, actualEnd})
}

// cut removes a range, potentially splitting one interval into two.
func (f *Filter) cut(start, end uint32) {
	list := *f
	idx := sort.Search(len(list), func(i int) bool {
		return list[i].End >= start
	})

	var fragments []Interval
	mergeEnd := idx

	for i := idx; i < len(list); i++ {
		target := list[i]
		if target.Start > end {
			break
		}

		// left fragment remains
		if target.Start < start {
			fragments = append(fragments, Interval{target.Start, start - 1})
		}
		// right fragment remains
		if target.End > end {
			fragments = append(fragments, Interval{end + 1, target.End})
		}
		mergeEnd = i + 1
	}

	*f = slices.Replace(list, idx, mergeEnd, fragments...)
}
