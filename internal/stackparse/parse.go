package stackparse

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Goroutine struct {
	ID          int
	State       string
	WaitMinutes int
	TopFrame    string
}

var (
	headerRe = regexp.MustCompile(`^goroutine (\d+) \[(.+)\]:$`)
	waitRe   = regexp.MustCompile(`(\d+) minutes?`)
)

func Parse(dump string) []Goroutine {
	blocks := strings.Split(strings.TrimSpace(dump), "\n\n")
	out := make([]Goroutine, 0, len(blocks))

	for _, block := range blocks {
		lines := strings.Split(block, "\n")
		if len(lines) < 2 {
			continue
		}

		m := headerRe.FindStringSubmatch(strings.TrimSpace(lines[0]))
		if m == nil {
			continue
		}

		id, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}

		rawState := m[2]
		state := rawState
		waitMin := 0
		if wm := waitRe.FindStringSubmatch(rawState); wm != nil {
			waitMin, _ = strconv.Atoi(wm[1])
			if idx := strings.Index(rawState, ","); idx != -1 {
				state = strings.TrimSpace(rawState[:idx])
			}
		}

		top := strings.TrimSpace(lines[1])
		if idx := strings.Index(top, "("); idx != -1 {
			top = top[:idx]
		}

		out = append(out, Goroutine{
			ID:          id,
			State:       state,
			WaitMinutes: waitMin,
			TopFrame:    top,
		})
	}

	return out
}

type HotPath struct {
	Frame string `json:"frame"`
	Count int    `json:"count"`
	Stuck bool   `json:"stuck"`
}

const stuckThresholdMinutes = 1

func Group(gs []Goroutine, stuckIDs map[int]bool) []HotPath {
	byFrame := map[string]*HotPath{}
	order := make([]string, 0, len(gs))

	for _, g := range gs {
		hp, ok := byFrame[g.TopFrame]
		if !ok {
			hp = &HotPath{Frame: g.TopFrame}
			byFrame[g.TopFrame] = hp
			order = append(order, g.TopFrame)
		}
		hp.Count++
		if g.WaitMinutes >= stuckThresholdMinutes || stuckIDs[g.ID] {
			hp.Stuck = true
		}
	}

	result := make([]HotPath, 0, len(order))
	for _, frame := range order {
		result = append(result, *byFrame[frame])
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}
