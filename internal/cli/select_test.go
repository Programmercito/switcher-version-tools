package cli

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Programmercito/switcher-version-tools/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func key(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func press(m tea.Model, msgs ...tea.Msg) selectModel {
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}
	return m.(selectModel)
}

func sampleItems() []Item {
	return []Item{
		{Label: "jdk8", Value: "jdk8"},
		{Label: "jdk17", Value: "jdk17", Current: true},
		{Label: "jdk21", Value: "jdk21"},
	}
}

func TestSelectStartsOnCurrent(t *testing.T) {
	m := newSelectModel("t", sampleItems(), false)
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.cursor)
	}
}

func TestSelectArrowsAndEnter(t *testing.T) {
	m := press(newSelectModel("t", sampleItems(), false), key(tea.KeyDown), key(tea.KeyEnter))
	if m.chosen == nil || m.chosen.Value != "jdk21" {
		t.Fatalf("chosen = %+v, want jdk21", m.chosen)
	}
}

func TestSelectWraps(t *testing.T) {
	m := press(newSelectModel("t", sampleItems(), false), key(tea.KeyHome), key(tea.KeyUp), key(tea.KeyEnter))
	if m.chosen == nil || m.chosen.Value != "jdk21" {
		t.Fatalf("chosen = %+v, want jdk21", m.chosen)
	}
}

func TestSelectVimKeys(t *testing.T) {
	m := press(newSelectModel("t", sampleItems(), false), runes("k"), key(tea.KeyEnter))
	if m.chosen == nil || m.chosen.Value != "jdk8" {
		t.Fatalf("chosen = %+v, want jdk8", m.chosen)
	}
}

func TestSelectCancel(t *testing.T) {
	for _, k := range []tea.KeyMsg{key(tea.KeyEsc), key(tea.KeyCtrlC), runes("q")} {
		m := press(newSelectModel("t", sampleItems(), false), k)
		if !errors.Is(m.err, ErrCancelled) {
			t.Fatalf("key %v: err = %v, want ErrCancelled", k, m.err)
		}
	}
}

func TestSelectBack(t *testing.T) {
	m := press(newSelectModel("t", sampleItems(), true), key(tea.KeyLeft))
	if !errors.Is(m.err, ErrBack) {
		t.Fatalf("left: err = %v, want ErrBack", m.err)
	}

	m = press(newSelectModel("t", sampleItems(), true), key(tea.KeyEnd), key(tea.KeyEnter))
	if !errors.Is(m.err, ErrBack) {
		t.Fatalf("back item: err = %v, want ErrBack", m.err)
	}

	m = press(newSelectModel("t", sampleItems(), false), key(tea.KeyLeft))
	if m.done {
		t.Fatal("left without allowBack must be ignored")
	}
}

func TestSelectFilter(t *testing.T) {
	m := press(newSelectModel("t", sampleItems(), true), runes("/"), runes("2"), runes("1"))
	if len(m.filtered) != 2 { // jdk21 + Volver
		t.Fatalf("filtered = %d, want 2", len(m.filtered))
	}
	m = press(m, key(tea.KeyEnter))
	if m.chosen == nil || m.chosen.Value != "jdk21" {
		t.Fatalf("chosen = %+v, want jdk21", m.chosen)
	}
}

func TestSelectFilterEscClears(t *testing.T) {
	m := press(newSelectModel("t", sampleItems(), false), runes("/"), runes("8"), key(tea.KeyEsc))
	if m.done || m.filtering || len(m.filtered) != 3 {
		t.Fatalf("esc in filter must clear it: %+v", m)
	}
}

func TestSelectFilterTypingJK(t *testing.T) {
	m := press(newSelectModel("t", sampleItems(), false), runes("/"), runes("j"), runes("k"))
	if m.filter != "jk" {
		t.Fatalf("filter = %q, want jk", m.filter)
	}
}

func TestSelectScrollsLongLists(t *testing.T) {
	items := make([]Item, 25)
	for i := range items {
		items[i] = Item{Label: "v", Value: "v"}
	}
	m := press(newSelectModel("t", items, false), key(tea.KeyEnd))
	if m.cursor != 24 || m.offset != 24-maxVisibleItems+1 {
		t.Fatalf("cursor=%d offset=%d", m.cursor, m.offset)
	}
}

func TestGroupByType(t *testing.T) {
	types, groups := groupByType([]config.Download{
		{Alias: "node20", Type: "node"},
		{Alias: "jdk21", Type: "java"},
		{Alias: "jdk17", Type: "java"},
	})
	if want := []string{"java", "node"}; !reflect.DeepEqual(types, want) {
		t.Fatalf("types = %v, want %v", types, want)
	}
	if want := []string{"jdk17", "jdk21"}; !reflect.DeepEqual(groups["java"], want) {
		t.Fatalf("java = %v, want %v", groups["java"], want)
	}
}

func TestPickInstalledEmpty(t *testing.T) {
	_, err := PickInstalled(&config.Config{}, "usar")
	if !errors.Is(err, ErrNoInstalls) {
		t.Fatalf("err = %v, want ErrNoInstalls", err)
	}
}

func TestPickInstalledSingleEntrySkipsPrompts(t *testing.T) {
	cfg := &config.Config{Downloads: []config.Download{{Alias: "php8.2", Type: "php"}}}
	alias, err := PickInstalled(cfg, "usar")
	if err != nil || alias != "php8.2" {
		t.Fatalf("alias=%q err=%v", alias, err)
	}
}
