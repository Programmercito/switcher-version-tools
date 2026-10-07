package cli

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/Programmercito/switcher-version-tools/internal/config"
)

// ErrNoInstalls indica que no hay instalaciones registradas para elegir.
var ErrNoInstalls = errors.New("no hay instalaciones registradas")

// IsInteractive indica si stdin y stdout son una terminal.
func IsInteractive() bool {
	return isCharDevice(os.Stdin) && isCharDevice(os.Stdout)
}

func isCharDevice(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// groupByType agrupa los alias por tipo y retorna los tipos ordenados.
func groupByType(downloads []config.Download) ([]string, map[string][]string) {
	groups := make(map[string][]string)
	for _, d := range downloads {
		groups[d.Type] = append(groups[d.Type], d.Alias)
	}
	types := make([]string, 0, len(groups))
	for t, aliases := range groups {
		sort.Strings(aliases)
		types = append(types, t)
	}
	sort.Strings(types)
	return types, groups
}

func versionsLabel(n int) string {
	if n == 1 {
		return "1 versión"
	}
	return fmt.Sprintf("%d versiones", n)
}

// PickInstalled deja elegir con las flechas el lenguaje y luego la versión
// instalada, y retorna el alias elegido. Se omite el primer paso si solo hay
// un lenguaje, y el segundo si ese lenguaje tiene una sola versión.
func PickInstalled(cfg *config.Config, action string) (string, error) {
	types, groups := groupByType(cfg.Downloads)
	if len(types) == 0 {
		return "", ErrNoInstalls
	}

	for {
		toolType := types[0]
		if len(types) > 1 {
			items := make([]Item, len(types))
			for i, t := range types {
				items[i] = Item{Label: t, Value: t, Desc: "(" + versionsLabel(len(groups[t])) + ")"}
			}
			it, err := Select("Lenguaje a "+action, items, false)
			if err != nil {
				return "", err
			}
			toolType = it.Value
		}

		aliases := groups[toolType]
		if len(aliases) == 1 {
			return aliases[0], nil
		}

		items := make([]Item, len(aliases))
		for i, a := range aliases {
			items[i] = Item{Label: a, Value: a, Current: cfg.Current[toolType] == a}
		}
		it, err := Select("Versión de "+toolType+" a "+action, items, len(types) > 1)
		if errors.Is(err, ErrBack) {
			continue
		}
		if err != nil {
			return "", err
		}
		return it.Value, nil
	}
}
