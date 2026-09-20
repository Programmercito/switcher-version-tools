package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/Programmercito/switcher-version-tools/internal/tool"
)

const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
)

var noColor = os.Getenv("NO_COLOR") != ""

func Printf(color, format string, a ...any) {
	if noColor {
		fmt.Printf(format, a...)
		return
	}
	fmt.Printf(color+format+Reset, a...)
}

func Println(color, s string) {
	Printf(color, "%s\n", s)
}

func Errorf(format string, a ...any)   { Printf(Red, format, a...) }
func Successf(format string, a ...any) { Printf(Green, format, a...) }
func Infof(format string, a ...any)    { Printf(Cyan, format, a...) }
func Warnf(format string, a ...any)    { Printf(Yellow, format, a...) }

func PrintLogo() {
	const logo = ` ____          _ _       _       _____           _ 
/ ___|_      _(_) |_ ___| |__   |_   _|__   ___ | |
\___ \ \ /\ / / | __/ __| '_ \    | |/ _ \ / _ \| |
 ___) \ V  V /| | || (__| | | |   | | (_) | (_) | |
|____/ \_/\_/ |_|\__\___|_| |_|   |_|\___/ \___/|_|`
	Printf(Yellow, "%s\n", logo)
	Printf(Cyan, "  switchtool — gestiona versiones de Java/Maven/Gradle/PHP/Go/Node\n\n")
}

func ShowHelp(version string) {
	Printf(Cyan, "Uso:\n")
	fmt.Println("  switchtool <tipo> <alias> <url|path_zip>  - Descargar o usar zip/tar.gz local e instalar")
	fmt.Println("  switchtool <alias>                         - Cambiar a una versión ya instalada")
	fmt.Println("  switchtool list | ls                       - Listar todas las instalaciones disponibles")
	fmt.Println("  switchtool remove <alias>                  - Eliminar un alias del registro y del disco")
	fmt.Println("  switchtool --version                       - Mostrar versión")
	fmt.Printf("\nTipos soportados: %s\n", strings.Join(tool.Supported(), ", "))
	fmt.Println("\nEjemplos:")
	fmt.Println("  switchtool java jdk17 https://.../jdk17.zip")
	fmt.Println("  switchtool php 8.1 C:\\Downloads\\php-8.1.zip")
	fmt.Println("  switchtool node v20 https://.../node-v20.tar.gz")
	if version != "" {
		Printf(Yellow, "\nVersión: %s\n", version)
	}
}

func PrintProgress(downloaded, total int64) {
	const barWidth = 20
	if total > 0 {
		progress := float64(downloaded) / float64(total) * 100
		filled := int(progress / 100 * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		fmt.Printf("\r%sDescargando... [%s] %.2f%%%s", Cyan, Green+bar+Reset+Cyan, progress, Reset)
	} else {
		fmt.Printf("\r%sDescargando... %s descargados%s", Cyan, formatBytes(downloaded), Reset)
	}
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit {
		div *= unit
		exp++
	}
	suffix := []string{"KB", "MB", "GB", "TB"}[exp]
	return fmt.Sprintf("%.2f %s", float64(n)/float64(div), suffix)
}

func PrintExtractProgress(current, total int, name string) {
	if total > 0 {
		fmt.Printf("\r%sDescomprimiendo... %d/%d%s", Cyan, current, total, Reset)
	} else {
		fmt.Printf("\r%sDescomprimiendo... %d%s", Cyan, current, Reset)
	}
}
