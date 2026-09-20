package tool

import (
	"os"
	"path/filepath"
)

// Tool describe cómo se configura una herramienta en el sistema.
type Tool struct {
	Name          string
	HomeVar       string
	ExtraHomeVars []string
	// BinRelative indica dónde están los ejecutables dentro del home.
	// "." significa que están en la raíz del paquete (PHP/Node en Windows).
	BinRelative string
	// Executable, si no está vacío, se usa para detectar el directorio raíz real.
	Executable string
}

// BinPath retorna la ruta que debe agregarse al PATH.
func (t Tool) BinPath(home string) string {
	if t.BinRelative == "." || t.BinRelative == "" {
		return home
	}
	return filepath.Join(home, t.BinRelative)
}

// Registry contiene las herramientas soportadas.
var Registry = map[string]Tool{
	"java": {
		Name:          "java",
		HomeVar:       "JAVA_HOME",
		ExtraHomeVars: []string{"JDK_HOME"},
		BinRelative:   "bin",
	},
	"maven": {
		Name:        "maven",
		HomeVar:     "MAVEN_HOME",
		BinRelative: "bin",
	},
	"gradle": {
		Name:        "gradle",
		HomeVar:     "GRADLE_HOME",
		BinRelative: "bin",
	},
	"php": {
		Name:        "php",
		HomeVar:     "PHP_HOME",
		BinRelative: ".",
		Executable:  "php.exe",
	},
	"go": {
		Name:        "go",
		HomeVar:     "GOROOT",
		BinRelative: "bin",
	},
	"node": {
		Name:        "node",
		HomeVar:     "NODE_HOME",
		BinRelative: ".",
	},
}

// Supported retorna los nombres de tipos soportados.
func Supported() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}

// IsValid indica si un tipo de herramienta está soportado.
func IsValid(name string) bool {
	_, ok := Registry[name]
	return ok
}

// Get obtiene una herramienta del registro.
func Get(name string) (Tool, bool) {
	t, ok := Registry[name]
	return t, ok
}

// DetectRoot detecta el directorio real de instalación dentro del target extraído.
func DetectRoot(tool Tool, targetDir string) (string, error) {
	if tool.Executable != "" {
		if pathExists(filepath.Join(targetDir, tool.Executable)) {
			return targetDir, nil
		}
		entries, err := os.ReadDir(targetDir)
		if err != nil {
			return "", err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				candidate := filepath.Join(targetDir, entry.Name())
				if pathExists(filepath.Join(candidate, tool.Executable)) {
					return candidate, nil
				}
			}
		}
		return targetDir, nil
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return "", err
	}
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(targetDir, entries[0].Name()), nil
	}
	return targetDir, nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
