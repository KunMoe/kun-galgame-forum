package renumber

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	HowCurated   = "curated"
	HowClaim     = "claim"
	HowUnchanged = "unchanged"
)

type Entry struct {
	OldID int64
	NewID int64
	How   string
}

type Map struct {
	byOld   map[int64]Entry
	order   []int64
	howN    map[string]int
	tsvRows int
}

func LoadMapFile(path string) (*Map, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open map: %w", err)
	}
	defer f.Close()
	return LoadMap(f)
}

func LoadMap(r io.Reader) (*Map, error) {
	m := &Map{
		byOld: make(map[int64]Entry),
		howN:  map[string]int{},
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			return nil, fmt.Errorf("map line %d: want old_id<TAB>new_id<TAB>how, got %d field(s)", lineNo, len(parts))
		}
		oldID, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		newID, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		how := strings.TrimSpace(parts[2])
		if err1 != nil {
			return nil, fmt.Errorf("map line %d: old_id: %w", lineNo, err1)
		}
		if err2 != nil {
			return nil, fmt.Errorf("map line %d: new_id: %w", lineNo, err2)
		}
		if oldID <= 0 || newID <= 0 {
			return nil, fmt.Errorf("map line %d: ids must be positive, got old_id=%d new_id=%d", lineNo, oldID, newID)
		}
		if how != HowCurated && how != HowClaim {
			return nil, fmt.Errorf("map line %d: unknown how %q", lineNo, how)
		}
		if _, ok := m.byOld[oldID]; ok {
			return nil, fmt.Errorf("map line %d: duplicated old_id %d", lineNo, oldID)
		}
		m.add(Entry{OldID: oldID, NewID: newID, How: how})
		m.tsvRows++
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read map: %w", err)
	}
	return m, nil
}

func (m *Map) Lookup(n int64) (int64, bool) {
	e, ok := m.byOld[n]
	if !ok {
		return 0, false
	}
	return e.NewID, true
}

func (m *Map) AddUnchanged(ids []int64) {
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := m.byOld[id]; ok {
			continue
		}
		m.add(Entry{OldID: id, NewID: id, How: HowUnchanged})
	}
}

func (m *Map) Entries() []Entry {
	out := make([]Entry, len(m.order))
	for i, id := range m.order {
		out[i] = m.byOld[id]
	}
	return out
}

func (m *Map) Len() int { return len(m.order) }

func (m *Map) HowCounts() map[string]int {
	out := make(map[string]int, len(m.howN))
	for k, v := range m.howN {
		out[k] = v
	}
	return out
}

func (m *Map) TSVRows() int { return m.tsvRows }

func (m *Map) add(e Entry) {
	m.byOld[e.OldID] = e
	m.order = append(m.order, e.OldID)
	m.howN[e.How]++
}
