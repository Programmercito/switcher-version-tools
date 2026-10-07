package cli

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Errores devueltos por Select.
var (
	ErrCancelled = errors.New("selección cancelada")
	ErrBack      = errors.New("volver")
)

const maxVisibleItems = 10

// Item es una opción del selector.
type Item struct {
	Label   string
	Desc    string
	Value   string
	Current bool
	back    bool
}

type selectModel struct {
	title     string
	items     []Item
	filtered  []int
	cursor    int
	offset    int
	filtering bool
	filter    string
	allowBack bool
	chosen    *Item
	err       error
	done      bool
}

func newSelectModel(title string, items []Item, allowBack bool) selectModel {
	all := make([]Item, len(items), len(items)+1)
	copy(all, items)
	if allowBack {
		all = append(all, Item{Label: "← Volver", back: true})
	}
	m := selectModel{title: title, items: all, allowBack: allowBack}
	m.refilter()
	for i, idx := range m.filtered {
		if m.items[idx].Current {
			m.cursor = i
			break
		}
	}
	m.clamp()
	return m
}

func (m *selectModel) refilter() {
	m.filtered = m.filtered[:0]
	q := strings.ToLower(m.filter)
	for i, it := range m.items {
		if it.back || q == "" || strings.Contains(strings.ToLower(it.Label), q) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor = 0
	m.offset = 0
}

func (m *selectModel) move(delta int) {
	if len(m.filtered) == 0 {
		return
	}
	m.cursor = (m.cursor + delta + len(m.filtered)) % len(m.filtered)
	m.clamp()
}

func (m *selectModel) clamp() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+maxVisibleItems {
		m.offset = m.cursor - maxVisibleItems + 1
	}
}

func (m *selectModel) quit(err error) (tea.Model, tea.Cmd) {
	m.err = err
	m.done = true
	return *m, tea.Quit
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch km.Type {
	case tea.KeyCtrlC:
		return m.quit(ErrCancelled)
	case tea.KeyUp:
		m.move(-1)
	case tea.KeyDown, tea.KeyTab:
		m.move(1)
	case tea.KeyHome:
		m.cursor = 0
		m.clamp()
	case tea.KeyEnd:
		m.cursor = max(len(m.filtered)-1, 0)
		m.clamp()
	case tea.KeyLeft:
		if m.allowBack && !m.filtering {
			return m.quit(ErrBack)
		}
	case tea.KeyEnter:
		if len(m.filtered) == 0 {
			return m, nil
		}
		it := m.items[m.filtered[m.cursor]]
		if it.back {
			return m.quit(ErrBack)
		}
		m.chosen = &it
		m.done = true
		return m, tea.Quit
	case tea.KeyEsc:
		if m.filtering {
			m.filtering = false
			m.filter = ""
			m.refilter()
			return m, nil
		}
		return m.quit(ErrCancelled)
	case tea.KeyBackspace:
		if m.filtering && m.filter != "" {
			r := []rune(m.filter)
			m.filter = string(r[:len(r)-1])
			m.refilter()
		}
	case tea.KeySpace:
		if m.filtering {
			m.filter += " "
			m.refilter()
		}
	case tea.KeyRunes:
		s := string(km.Runes)
		if m.filtering {
			m.filter += s
			m.refilter()
			return m, nil
		}
		switch s {
		case "k":
			m.move(-1)
		case "j":
			m.move(1)
		case "/":
			m.filtering = true
		case "q":
			return m.quit(ErrCancelled)
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	if m.done {
		if m.chosen != nil {
			return successStyle.Render("✓ ") + m.title + " " + infoStyle.Render(m.chosen.Label) + "\n"
		}
		return ""
	}

	return m.renderBox()
}

var (
	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1B1D17")).Background(monoColor)
	selectBoxStyle   = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(monoColor).
				Padding(0, 2)
)

const minBoxWidth = 50

func (m selectModel) renderBox() string {
	hint := "↑↓ mover   ⏎ elegir   / filtrar   esc salir"
	if m.allowBack {
		hint = "↑↓ mover   ⏎ elegir   ← volver   / filtrar   esc salir"
	}

	labelW := 0
	for _, it := range m.items {
		labelW = max(labelW, lipgloss.Width(it.Label))
	}

	rows := make([]string, 0, maxVisibleItems)
	plain := make([]string, 0, maxVisibleItems)
	end := min(m.offset+maxVisibleItems, len(m.filtered))
	for i := m.offset; i < end; i++ {
		it := m.items[m.filtered[i]]
		pad := strings.Repeat(" ", labelW-lipgloss.Width(it.Label))
		desc, actual := "", ""
		if it.Desc != "" {
			desc = "  " + it.Desc
		}
		if it.Current {
			actual = "  ● actual"
		}
		plain = append(plain, "  "+it.Label+pad+desc+actual)
		if i == m.cursor {
			rows = append(rows, "")
			continue
		}
		row := "  " + it.Label + pad
		if desc != "" {
			row += mutedStyle.Render(desc)
		}
		if actual != "" {
			row += successStyle.Render(actual)
		}
		rows = append(rows, row)
	}

	width := max(minBoxWidth, lipgloss.Width(m.title)+4, lipgloss.Width(hint))
	for _, p := range plain {
		width = max(width, lipgloss.Width(p)+2)
	}

	for k := range rows {
		if m.offset+k == m.cursor {
			rows[k] = selectedRowStyle.Width(width).Render("❯ " + strings.TrimPrefix(plain[k], "  "))
		}
	}

	var b strings.Builder
	b.WriteString(infoStyle.Render("▍") + titleStyle.Render(m.title) + "\n")
	if m.filtering {
		b.WriteString(mutedStyle.Render("filtro › ") + m.filter + infoStyle.Render("█") + "\n")
	}
	b.WriteString("\n")
	if len(rows) == 0 {
		b.WriteString(mutedStyle.Render("  Sin resultados") + "\n")
	}
	b.WriteString(strings.Join(rows, "\n"))
	if len(m.filtered) > maxVisibleItems {
		b.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("  %d/%d", m.cursor+1, len(m.filtered))))
	}
	b.WriteString("\n\n" + mutedStyle.Render(hint))

	return selectBoxStyle.Width(width+4).Render(b.String()) + "\n"
}

// Select muestra una lista navegable con las flechas y retorna el item elegido.
// Retorna ErrBack si el usuario elige volver y ErrCancelled si cancela.
func Select(title string, items []Item, allowBack bool) (Item, error) {
	if len(items) == 0 {
		return Item{}, errors.New("no hay opciones para elegir")
	}
	final, err := tea.NewProgram(newSelectModel(title, items, allowBack)).Run()
	if err != nil {
		return Item{}, err
	}
	m := final.(selectModel)
	if m.err != nil {
		return Item{}, m.err
	}
	if m.chosen == nil {
		return Item{}, ErrCancelled
	}
	return *m.chosen, nil
}

// Confirm pregunta Sí/No con el mismo selector. El valor por defecto es No.
func Confirm(title string) (bool, error) {
	it, err := Select(title, []Item{
		{Label: "No", Value: "no"},
		{Label: "Sí", Value: "yes"},
	}, false)
	if err != nil {
		return false, err
	}
	return it.Value == "yes", nil
}
