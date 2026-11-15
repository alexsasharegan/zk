package box

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func keys(fm map[string]any) string {
	ks := make([]string, 0, len(fm))
	for k := range fm {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return strings.Join(ks, " ")
}

func pkmMapping(t *testing.T) Mapping {
	t.Helper()
	s, err := Load(vault(t, "box/zk.toml", pkmLike))
	if err != nil {
		t.Fatal(err)
	}
	return s.Mapping
}

var (
	day      = time.Date(2026, 11, 13, 0, 0, 0, 0, time.Local)
	instant  = time.Date(2026, 11, 13, 0, 16, 1, 0, time.Local)
	sampleCd = Card{Kind: "claim", Key: "2a", Level: 2, Sort: "00000002/00000001", Status: "linked", AuthoredBy: "agent"}
)

func TestDefaultMappingWritesOnlyZkNames(t *testing.T) {
	m := DefaultMapping()
	fm := map[string]any{"aliases": []string{"Grassé's coinage"}, "tags": []string{"zk/2/a"}, "mine": "kept"}
	c := sampleCd
	c.Created = instant
	m.PutCard(fm, c)
	want := "aliases mine tags zk_authored_by zk_class zk_created zk_key zk_kind zk_level zk_sort zk_status"
	if got := keys(fm); got != want {
		t.Fatalf("keys:\n got %s\nwant %s", got, want)
	}
	if fm["mine"] != "kept" || fm["zk_class"] != "card" {
		t.Errorf("fm: %v", fm)
	}
	if got := fm["zk_created"].(Date).String(); got != "2026-11-13T00:16:01" {
		t.Errorf("created written as %q", got)
	}
	back, err := m.Card(fm)
	if err != nil || back != c {
		t.Fatalf("round trip: %+v, %v", back, err)
	}
}

func TestMappedCardRoundTrip(t *testing.T) {
	m := pkmMapping(t)
	fm := map[string]any{"aliases": []string{"Grassé's coinage"}, "tags": []string{"zk/2/a"}}
	c := sampleCd
	c.Created = day
	m.PutCard(fm, c)
	want := "aliases authored_by created_at kind status tags type zk_key zk_level zk_sort"
	if got := keys(fm); got != want {
		t.Fatalf("keys:\n got %s\nwant %s", got, want)
	}
	if fm["type"] != "zk-card" || fm["kind"] != "claim" {
		t.Errorf("class and kind: %v, %v", fm["type"], fm["kind"])
	}
	if got := fm["created_at"].(Date).String(); got != "2026-11-13" {
		t.Errorf("created_at written as %q, want a date", got)
	}
	back, err := m.Card(fm)
	if err != nil || back != c {
		t.Fatalf("round trip: %+v, %v", back, err)
	}
}

func TestMappedSourceRoundTrip(t *testing.T) {
	m := pkmMapping(t)
	s := Source{
		Title:      "What is the point of libraries?",
		Author:     "A. Writer",
		Published:  time.Date(2025, 8, 20, 0, 0, 0, 0, time.Local),
		Provenance: "https://example.com/libraries",
		Via:        "a read-later app",
		Form:       "article",
		Status:     "extracted",
		Created:    day,
		Extracted:  []string{"65", "65a"},
		Series:     "Notes",
	}
	fm := map[string]any{}
	m.PutSource(fm, s)
	want := "author created_at published series status title type zk_extracted zk_form zk_provenance zk_via"
	if got := keys(fm); got != want {
		t.Fatalf("keys:\n got %s\nwant %s", got, want)
	}
	back, err := m.Source(fm)
	if err != nil || !reflect.DeepEqual(back, s) {
		t.Fatalf("round trip:\n got %+v\nwant %+v\n%v", back, s, err)
	}
}

func TestEmptyValuesRemoveFields(t *testing.T) {
	m := DefaultMapping()
	fm := map[string]any{"zk_authored_by": "agent", "zk_level": 3}
	m.PutCard(fm, Card{Kind: "idea", Key: "1"})
	if _, ok := fm["zk_authored_by"]; ok {
		t.Error("an empty authored_by should remove the field")
	}
	if _, ok := fm["zk_level"]; ok {
		t.Error("a zero level should remove the field")
	}
}

// TestReadsParsedValues covers values as a YAML parser hands them over:
// dates as strings or time.Time, an unquoted key as a number, lists as []any.
func TestReadsParsedValues(t *testing.T) {
	m := pkmMapping(t)
	fm := map[string]any{
		"type":       "zk-card",
		"kind":       "idea",
		"zk_key":     7,
		"zk_level":   1,
		"status":     "raw",
		"created_at": "2026-11-13T00:16:01",
	}
	c, err := m.Card(fm)
	if err != nil {
		t.Fatal(err)
	}
	if c.Key != "7" || !c.Created.Equal(instant) {
		t.Errorf("got %+v", c)
	}

	src := map[string]any{"type": "zk-source", "created_at": day, "zk_extracted": []any{"1", 2}}
	s, err := m.Source(src)
	if err != nil || !s.Created.Equal(day) || !reflect.DeepEqual(s.Extracted, []string{"1", "2"}) {
		t.Fatalf("got %+v, %v", s, err)
	}
}

func TestWrongClassAndBadValues(t *testing.T) {
	m := pkmMapping(t)
	if _, err := m.Card(map[string]any{"type": "zk-source"}); !errors.Is(err, ErrWrongClass) {
		t.Errorf("source read as card: %v", err)
	}
	if _, err := m.Source(map[string]any{"type": "daily"}); !errors.Is(err, ErrWrongClass) {
		t.Errorf("daily note read as source: %v", err)
	}
	_, err := m.Card(map[string]any{"type": "zk-card", "zk_level": "two", "created_at": "yesterday", "kind": []any{"x"}})
	if err == nil {
		t.Fatal("want type errors")
	}
	for _, field := range []string{"zk_level", "created_at", "kind"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("error %q does not name %s", err, field)
		}
	}
}

func TestClassOf(t *testing.T) {
	m := pkmMapping(t)
	if c, ok := m.ClassOf(map[string]any{"type": "zk-source"}); !ok || c != ClassSource {
		t.Errorf("got %q, %v", c, ok)
	}
	if _, ok := m.ClassOf(map[string]any{"type": "daily"}); ok {
		t.Error("a daily note is not a box note")
	}
}
