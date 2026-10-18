package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Programmercito/switcher-version-tools/internal/config"
	"github.com/Programmercito/switcher-version-tools/internal/tool"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
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
		progress.WithGradient("#3F6B1C", "#C8FF4D"),
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
	fmt.Println(subtitleStyle.Render("  developer environment version manager"))
	fmt.Println()
}

// ShowHelp imprime la ayuda dentro de un recuadro.
func ShowHelp(version string) {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("COMANDOS") + "\n")
	b.WriteString("  switchtool <tipo> <alias> <url|path_zip>  Descargar o usar zip/tar.gz local e instalar\n")
	b.WriteString("  switchtool | switchtool use                Elegir herramienta y versión con las flechas\n")
	b.WriteString("  switchtool <alias>                         Cambiar a una versión ya instalada\n")
	b.WriteString("  switchtool list | ls                       Listar todas las instalaciones disponibles\n")
	b.WriteString("  switchtool remove <alias>                  Eliminar un alias del registro y del disco\n")
	b.WriteString("  switchtool remove                          Elegir con las flechas qué eliminar\n")
	b.WriteString("  switchtool --version                       Mostrar versión\n\n")
	b.WriteString(supportedToolsHelp())
	b.WriteString("\n" + TitleStyle.Render("EJEMPLOS") + "\n")
	b.WriteString("  switchtool java jdk17 https://.../jdk17.zip\n")
	b.WriteString("  switchtool php 8.1 C:\\Downloads\\php-8.1.zip\n")
	b.WriteString("  switchtool node v20 https://.../node-v20.tar.gz\n")
	b.WriteString("  switchtool pnpm v12.11.2 <url|zip>\n")
	if version != "" {
		b.WriteString("\n" + mutedStyle.Render("Versión: "+version))
	}

	fmt.Println(boxStyle.Render(b.String()))
}

func supportedToolsHelp() string {
	names := tool.Supported()
	sort.Strings(names)

	var b strings.Builder
	b.WriteString(TitleStyle.Render("HERRAMIENTAS SOPORTADAS") + "\n")
	for _, name := range names {
		t, ok := tool.Get(name)
		if !ok {
			continue
		}

		label := name
		switch name {
		case "java":
			label = "Java (JDK)"
		case "gradle":
			label = "Gradle"
		case "maven":
			label = "Maven"
		case "go":
			label = "Go"
		case "node":
			label = "Node.js"
		case "php":
			label = "PHP"
		}

		homeVars := append([]string{t.HomeVar}, t.ExtraHomeVars...)
		fmt.Fprintf(
			&b,
			"  %s %-14s %s\n",
			SuccessStyle.Render("◆"),
			TitleStyle.Render(fmt.Sprintf("%-14s", label)),
			MutedStyle.Render(strings.Join(homeVars, ", ")),
		)
	}
	return b.String()
}

// printStyled aplica el estilo sin incluir los saltos de línea finales:
// lipgloss rellena con espacios las líneas vacías y desalinea la salida siguiente.
func printStyled(style lipgloss.Style, format string, a ...any) {
	text := fmt.Sprintf(format, a...)
	trimmed := strings.TrimRight(text, "\n")
	fmt.Print(style.Render(trimmed) + text[len(trimmed):])
}

// Errorf imprime un mensaje de error.
func Errorf(format string, a ...any) {
	printStyled(errorStyle, "✗ "+format, a...)
}

// Successf imprime un mensaje de éxito.
func Successf(format string, a ...any) {
	printStyled(successStyle, "✓ "+format, a...)
}

// Infof imprime un mensaje informativo.
func Infof(format string, a ...any) {
	printStyled(infoStyle, "→ "+format, a...)
}

// Warnf imprime una advertencia.
func Warnf(format string, a ...any) {
	printStyled(warnStyle, "⚡ "+format, a...)
}

// Mutedf imprime texto secundario.
func Mutedf(format string, a ...any) {
	printStyled(mutedStyle, format, a...)
}

// PrintDownloads muestra las instalaciones agrupadas por herramienta.
func PrintDownloads(cfg *config.Config) {
	types, groups := downloadsByType(cfg.Downloads)
	Infof("Instalaciones disponibles:\n")

	for i, toolType := range types {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("  %s %s\n",
			TitleStyle.Render(toolDisplayName(toolType)),
			MutedStyle.Render("· "+versionsLabel(len(groups[toolType]))),
		)
		for _, d := range groups[toolType] {
			label := d.Alias
			details := ""
			if d.Version != "" {
				label = d.Version
				details = " · " + MutedStyle.Render("alias "+d.Alias)
			}
			marker := ""
			if current, ok := cfg.Current[d.Type]; ok && current == d.Alias {
				marker = " " + SuccessStyle.Render("✓ actual")
			}
			fmt.Printf("    • %s%s%s\n", label, details, marker)
		}
	}
}

func toolDisplayName(name string) string {
	switch name {
	case "java":
		return "Java (JDK)"
	case "node":
		return "Node.js"
	case "php":
		return "PHP"
	case "go":
		return "Go"
	default:
		return strings.ToUpper(name[:1]) + name[1:]
	}
}

// PrintDownloadProgress muestra la barra de progreso o el spinner.
func (ui *UI) PrintDownloadProgress(downloaded, total int64) {
	if total > 0 {
		pct := min(float64(downloaded)/float64(total), 1)
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
