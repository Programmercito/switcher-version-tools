package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"github.com/Programmercito/switcher-version-tools/internal/tool"
)

// Tono verdoso monocromático.
var (
	monoColor     = lipgloss.Color("#A6E22E")
	monoDim       = lipgloss.Color("#7E8446")
	TitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(monoColor)
	SubtitleStyle = lipgloss.NewStyle().Foreground(monoDim)
	SuccessStyle  = lipgloss.NewStyle().Foreground(monoColor)
	ErrorStyle    = lipgloss.NewStyle().Bold(true).Foreground(monoColor)
	InfoStyle     = lipgloss.NewStyle().Foreground(monoColor)
	WarnStyle     = lipgloss.NewStyle().Foreground(monoColor)
	MutedStyle    = lipgloss.NewStyle().Foreground(monoDim)
	BoxStyle      = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(monoColor).
			Padding(1, 2).
			MarginTop(1)

	titleStyle    = TitleStyle
	subtitleStyle = SubtitleStyle
	successStyle  = SuccessStyle
	errorStyle    = ErrorStyle
	infoStyle     = InfoStyle
	warnStyle     = WarnStyle
	mutedStyle    = MutedStyle
	boxStyle      = BoxStyle
)

const logo = ` ____          _ _       _       _____           _ 
/ ___|_      _(_) |_ ___| |__   |_   _|__   ___ | |
\___ \ \ /\ / / | __/ __| '_ \    | |/ _ \ / _ \| |
 ___) \ V  V /| | || (__| | | |   | | (_) | (_) | |
|____/ \_/\_/ |_|\__\___|_| |_|   |_|\___/ \___/|_|`

// UI agrupa componentes interactivos de la interfaz.
type UI struct {
	progress     progress.Model
	spinnerFrame int
}

// NewUI crea una nueva interfaz con estilos y componentes listos.
func NewUI() *UI {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)
	return &UI{progress: p}
}

// PrintLogo muestra el logo estático en tono verdoso.
func PrintLogo() {
	fmt.Println()
	for _, line := range strings.Split(logo, "\n") {
		fmt.Println(titleStyle.Render(line))
	}
	fmt.Println(subtitleStyle.Render("  switchtool — gestiona versiones de Java/Maven/Gradle/PHP/Go/Node"))
	fmt.Println()
}

// ShowHelp imprime la ayuda dentro de un recuadro.
func ShowHelp(version string) {
	var b strings.Builder
	b.WriteString(infoStyle.Render("Uso") + "\n")
	b.WriteString("  switchtool <tipo> <alias> <url|path_zip>  Descargar o usar zip/tar.gz local e instalar\n")
	b.WriteString("  switchtool <alias>                         Cambiar a una versión ya instalada\n")
	b.WriteString("  switchtool list | ls                       Listar todas las instalaciones disponibles\n")
	b.WriteString("  switchtool remove <alias>                  Eliminar un alias del registro y del disco\n")
	b.WriteString("  switchtool --version                       Mostrar versión\n\n")
	b.WriteString("Tipos soportados: " + strings.Join(tool.Supported(), ", ") + "\n\n")
	b.WriteString(warnStyle.Render("Ejemplos") + "\n")
	b.WriteString("  switchtool java jdk17 https://.../jdk17.zip\n")
	b.WriteString("  switchtool php 8.1 C:\\Downloads\\php-8.1.zip\n")
	b.WriteString("  switchtool node v20 https://.../node-v20.tar.gz\n")
	if version != "" {
		b.WriteString("\n" + mutedStyle.Render("Versión: "+version))
	}

	fmt.Println(boxStyle.Render(b.String()))
}

// Errorf imprime un mensaje de error.
func Errorf(format string, a ...any) {
	fmt.Printf(errorStyle.Render("✗ "+format), a...)
}

// Successf imprime un mensaje de éxito.
func Successf(format string, a ...any) {
	fmt.Printf(successStyle.Render("✓ "+format), a...)
}

// Infof imprime un mensaje informativo.
func Infof(format string, a ...any) {
	fmt.Printf(infoStyle.Render("→ "+format), a...)
}

// Warnf imprime una advertencia.
func Warnf(format string, a ...any) {
	fmt.Printf(warnStyle.Render("⚡ "+format), a...)
}

// Mutedf imprime texto secundario.
func Mutedf(format string, a ...any) {
	fmt.Printf(mutedStyle.Render(format), a...)
}

// PrintDownloadProgress muestra la barra de progreso o el spinner.
func (ui *UI) PrintDownloadProgress(downloaded, total int64) {
	if total > 0 {
		pct := float64(downloaded) / float64(total)
		fmt.Printf("\r%s %s", infoStyle.Render("Descargando"), ui.progress.ViewAs(pct))
	} else {
		frame := ui.nextSpinner()
		fmt.Printf("\r%s %s %s descargados", infoStyle.Render("Descargando"), frame, formatBytes(downloaded))
	}
}

// PrintExtractProgress muestra el progreso de extracción.
func (ui *UI) PrintExtractProgress(current, total int, name string) {
	frame := ui.nextSpinner()
	if total > 0 {
		fmt.Printf("\r%s %s %d/%d", infoStyle.Render("Descomprimiendo"), frame, current, total)
	} else {
		fmt.Printf("\r%s %s %d", infoStyle.Render("Descomprimiendo"), frame, current)
	}
}

// PrintDownloadComplete limpia la línea de progreso.
func (ui *UI) PrintDownloadComplete() {
	fmt.Println()
	fmt.Println(successStyle.Render("✓ Descarga completada"))
}

// PrintExtractComplete limpia la línea de progreso.
func (ui *UI) PrintExtractComplete() {
	fmt.Println()
	fmt.Println(successStyle.Render("✓ Descompresión completada"))
}

func (ui *UI) nextSpinner() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	f := frames[ui.spinnerFrame%len(frames)]
	ui.spinnerFrame++
	return lipgloss.NewStyle().Foreground(monoColor).Render(f)
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

// ReloadNotice imprime la nota final sobre refrescar la terminal.
func ReloadNotice() {
	fmt.Println()
	fmt.Println(warnStyle.Render("IMPORTANTE:") + " Para que los cambios surtan efecto en la terminal actual:")
	fmt.Println("  1. Cerrá y abrí la terminal.")
	fmt.Println("  2. O ejecutá: " + infoStyle.Render("refreshenv") + mutedStyle.Render(" (si tenés Chocolatey)."))
}
