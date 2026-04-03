package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func printLogo() {
	fmt.Printf("%s", Yellow)
	fmt.Println(" ____          _ _       _       _____           _     ")
	fmt.Println("/ ___|_      _(_) |_ ___| |__   |_   _|__   ___ | |___ ")
	fmt.Println("\\___ \\ \\ /\\ / / | __/ __| '_ \\    | |/ _ \\ / _ \\| / __|")
	fmt.Println(" ___) \\ V  V /| | || (__| | | |   | | (_) | (_) | \\__ \\")
	fmt.Println("|____/ \\_/\\_/ |_|\\__\\___|_| |_|   |_|\\___/ \\___/|_|___/")
	fmt.Printf("%s\n", Reset)
	fmt.Printf("%s  switch-tools — gestiona versiones de Java/Maven/Gradle/PHP/Go/Node%s\n\n", Cyan, Reset)
}

type Download struct {
	Alias string `json:"alias"`
	Type  string `json:"type"`
}

type Config struct {
	Downloads []Download        `json:"downloads"`
	Current   map[string]string `json:"current"`
}

type progressWriter struct {
	total      int64
	downloaded int64
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.downloaded += int64(n)
	if pw.total > 0 {
		progress := float64(pw.downloaded) / float64(pw.total) * 100
		barWidth := 20
		filled := int(progress / 100 * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		fmt.Printf("\r%sDescargando... [%s] %.2f%%%s", Cyan, Green+bar+Reset+Cyan, progress, Reset)
	}
	return n, nil
}

func main() {
	printLogo()
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(1)
	}

	arg1 := os.Args[1]

	// Comandos de un solo argumento
	if len(os.Args) == 2 {
		if arg1 == "list" || arg1 == "ist" || arg1 == "ls" {
			listCurrent()
			return
		} else if arg1 == "help" || arg1 == "-h" || arg1 == "--help" {
			showHelp()
			return
		} else {
			// Modo cambio: <alias>
			switchToAlias(arg1)
		}
		return
	}

	// Comandos de tres argumentos: <tipo> <alias> <url_o_path>
	if len(os.Args) == 4 {
		tipo := os.Args[1]
		alias := os.Args[2]
		source := os.Args[3]

		validTypes := map[string]bool{
			"java": true, "maven": true, "gradle": true,
			"php": true, "go": true, "node": true,
		}

		if !validTypes[tipo] {
			fmt.Printf("%sError: Tipo '%s' no válido. Soportados: java, maven, gradle, php, go, node.%s\n", Red, tipo, Reset)
			os.Exit(1)
		}

		downloadAndSet(source, tipo, alias)
	} else {
		fmt.Printf("%sError: Número de parámetros incorrecto.%s\n\n", Red, Reset)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf("%sUso:%s\n", Cyan, Reset)
	fmt.Println("Recuerda que para instalar una nueva versión pasamos el tipo, el alias y al final la URL o PATH local.")
	fmt.Println("Esto permite que sea muy fácil repetir comandos cambiando solo lo último.")
	fmt.Println("  switch-tools <tipo> <alias> <url|path_zip> - Descargar o usar zip/tar.gz local e instalar")
	fmt.Println("  switch-tools <alias>                       - Cambiar a una versión ya instalada")
	fmt.Println("  switch-tools list | ls                     - Listar todas las instalaciones disponibles")
	fmt.Println("\nTipos soportados: java, maven, gradle, php, go, node")
	fmt.Println("\nEjemplos:")
	fmt.Println("  switch-tools java jdk17 https://.../jdk17.zip")
	fmt.Println("  switch-tools php 8.1 C:\\Downloads\\php-8.1.zip")
	fmt.Println("  switch-tools node v20 https://.../node-v20.tar.gz")
}

func downloadAndSet(urlStr, tipo, alias string) {
	fmt.Printf("%s🚀 Iniciando instalación de %s (%s)%s\n", Yellow, alias, tipo, Reset)
	fmt.Printf("%sURL: %s%s\n", Cyan, urlStr, Reset)

	// Verificar si el directorio ya existe
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("%sError al obtener home: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	targetDir := filepath.Join(homeDir, "switchjdk", alias)
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("%sError: El directorio %s ya existe. No se puede sobrescribir.%s\n", Red, targetDir, Reset)
		os.Exit(1)
	}

	// Descargar o usar archivo local
	var tempFile string
	if strings.HasPrefix(urlStr, "http") || strings.HasPrefix(urlStr, "https") {
		// Validar URL
		_, err := url.ParseRequestURI(urlStr)
		if err != nil {
			// Si no es URL válida, quizás es un path local relativo que no encontramos antes
			absPath, errAbs := filepath.Abs(urlStr)
			if errAbs == nil {
				if _, errStat := os.Stat(absPath); errStat == nil {
					tempFile = absPath
					goto skipDownload
				}
			}
			fmt.Printf("%sError: La URL o PATH proporcionado no es válido o no existe: %s%s\n", Red, urlStr, Reset)
			os.Exit(1)
		}

		tempDir := os.TempDir()
		tempFile = filepath.Join(tempDir, "download.zip")
		out, err := os.Create(tempFile)
		if err != nil {
			fmt.Printf("Error al crear archivo temp: %v\n", err)
			os.Exit(1)
		}
		defer out.Close()
		defer os.Remove(tempFile)

		resp, err := http.Get(urlStr)
		if err != nil {
			fmt.Printf("Error al descargar: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf("Error: Código de estado %d\n", resp.StatusCode)
			os.Exit(1)
		}

		pw := &progressWriter{total: resp.ContentLength}
		_, err = io.Copy(out, io.TeeReader(resp.Body, pw))
		if err != nil {
			fmt.Printf("Error al escribir archivo: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\nDescarga completada.")
	} else {
		// Archivo local
		absPath, err := filepath.Abs(urlStr)
		if err != nil {
			fmt.Printf("%sError al obtener ruta absoluta: %v%s\n", Red, err, Reset)
			os.Exit(1)
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Printf("%sError: Archivo local no encontrado en %s%s\n", Red, absPath, Reset)
			os.Exit(1)
		}
		tempFile = absPath
	}

skipDownload:

	// Descomprimir según formato
	if strings.HasSuffix(strings.ToLower(tempFile), ".zip") {
		r, err := zip.OpenReader(tempFile)
		if err != nil {
			fmt.Printf("%sError al abrir zip: %v%s\n", Red, err, Reset)
			os.Exit(1)
		}
		defer r.Close()

		totalFiles := len(r.File)
		extracted := 0
		for _, f := range r.File {
			fmt.Printf("\r%sDescomprimiendo... %d/%d%s", Cyan, extracted, totalFiles, Reset)
			fpath := filepath.Join(targetDir, f.Name)

			if !strings.HasPrefix(fpath, filepath.Clean(targetDir)+string(os.PathSeparator)) {
				fmt.Printf("%sError: Archivo zip malicioso%s\n", Red, Reset)
				os.Exit(1)
			}

			if f.FileInfo().IsDir() {
				os.MkdirAll(fpath, os.ModePerm)
				continue
			}

			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				fmt.Printf("%sError al crear directorio: %v%s\n", Red, err, Reset)
				os.Exit(1)
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				fmt.Printf("%sError al crear archivo: %v%s\n", Red, err, Reset)
				os.Exit(1)
			}

			rc, err := f.Open()
			if err != nil {
				outFile.Close()
				fmt.Printf("%sError al abrir archivo en zip: %v%s\n", Red, err, Reset)
				os.Exit(1)
			}

			_, err = io.Copy(outFile, rc)
			outFile.Close()
			rc.Close()
			if err != nil {
				fmt.Printf("%sError al copiar archivo: %v%s\n", Red, err, Reset)
				os.Exit(1)
			}
			extracted++
		}
		fmt.Printf("%s\nDescompresión completada. Archivos en %s%s\n", Green, targetDir, Reset)
	} else if strings.HasSuffix(strings.ToLower(tempFile), ".tar.gz") || strings.HasSuffix(strings.ToLower(tempFile), ".tgz") {
		file, err := os.Open(tempFile)
		if err != nil {
			fmt.Printf("%sError al abrir archivo: %v%s\n", Red, err, Reset)
			os.Exit(1)
		}
		defer file.Close()

		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			fmt.Printf("%sError al descomprimir gzip: %v%s\n", Red, err, Reset)
			os.Exit(1)
		}
		defer gzipReader.Close()

		tarReader := tar.NewReader(gzipReader)

		// Contar archivos
		totalFiles := 0
		for {
			_, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Printf("%sError al leer tar: %v%s\n", Red, err, Reset)
				os.Exit(1)
			}
			totalFiles++
		}

		// Reset
		file.Seek(0, 0)
		gzipReader, _ = gzip.NewReader(file)
		tarReader = tar.NewReader(gzipReader)

		extracted := 0
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Printf("%sError al leer tar: %v%s\n", Red, err, Reset)
				os.Exit(1)
			}

			fpath := filepath.Join(targetDir, header.Name)

			if !strings.HasPrefix(fpath, filepath.Clean(targetDir)+string(os.PathSeparator)) {
				fmt.Printf("%sError: Archivo tar malicioso%s\n", Red, Reset)
				os.Exit(1)
			}

			switch header.Typeflag {
			case tar.TypeDir:
				os.MkdirAll(fpath, os.FileMode(header.Mode))
			case tar.TypeReg:
				os.MkdirAll(filepath.Dir(fpath), 0755)
				outFile, err := os.Create(fpath)
				if err != nil {
					fmt.Printf("%sError al crear archivo: %v%s\n", Red, err, Reset)
					os.Exit(1)
				}
				_, err = io.Copy(outFile, tarReader)
				outFile.Close()
				if err != nil {
					fmt.Printf("%sError al copiar archivo: %v%s\n", Red, err, Reset)
					os.Exit(1)
				}
			}
			extracted++
			fmt.Printf("\r%sDescomprimiendo... %d/%d%s", Cyan, extracted, totalFiles, Reset)
		}
		fmt.Printf("%s\nDescompresión completada. Archivos en %s%s\n", Green, targetDir, Reset)
	} else {
		fmt.Printf("%sError: Formato de archivo no soportado. Solo .zip, .tar.gz y .tgz%s\n", Red, Reset)
		os.Exit(1)
	}

	// Detección compatible: si hay una subcarpeta, entramos (como antes).
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		fmt.Printf("%sError al leer directorio: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}

	var subDir string
	for _, entry := range entries {
		if entry.IsDir() {
			subDir = entry.Name()
			break
		}
	}

	actualHome := targetDir
	if subDir != "" {
		actualHome = filepath.Join(targetDir, subDir)
	}

	fmt.Printf("%sDirectorio principal detectado: %s%s\n", Green, actualHome, Reset)

	// Guardar en configuración
	configDir := filepath.Join(homeDir, "switchjdk")
	configFile := filepath.Join(configDir, "downloads.json")
	var config Config
	if data, err := os.ReadFile(configFile); err == nil {
		json.Unmarshal(data, &config)
	}
	if config.Current == nil {
		config.Current = make(map[string]string)
	}
	found := false
	for _, d := range config.Downloads {
		if d.Alias == alias {
			found = true
			break
		}
	}
	if !found {
		config.Downloads = append(config.Downloads, Download{Alias: alias, Type: tipo})
		data, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile(configFile, data, 0644)
		fmt.Printf("%sAlias '%s' agregado a la configuración.%s\n", Green, alias, Reset)
	} else {
		fmt.Printf("%sAlias '%s' ya existe en la configuración.%s\n", Yellow, alias, Reset)
	}

	setEnvVars(tipo, actualHome, alias)
}

func switchToAlias(alias string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("%sError al obtener home: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	configFile := filepath.Join(homeDir, "switchjdk", "downloads.json")
	var config Config
	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("%sError al leer configuración: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	json.Unmarshal(data, &config)
	if config.Current == nil {
		config.Current = make(map[string]string)
	}

	for _, d := range config.Downloads {
		if d.Alias == alias {
			// Detectar path con lógica compatible
			targetDir := filepath.Join(homeDir, "switchjdk", alias)
			entries, err := os.ReadDir(targetDir)
			if err != nil {
				fmt.Printf("%sError al leer directorio: %v%s\n", Red, err, Reset)
				os.Exit(1)
			}
			var subDir string
			for _, entry := range entries {
				if entry.IsDir() {
					subDir = entry.Name()
					break
				}
			}
			actualPath := targetDir
			if subDir != "" {
				actualPath = filepath.Join(targetDir, subDir)
			}
			fmt.Printf("%s🔄 Cambiando a alias '%s' (tipo: %s)...%s\n", Yellow, alias, d.Type, Reset)
			setEnvVars(d.Type, actualPath, alias)
			fmt.Printf("%s✅ Listo: ahora se está usando '%s' para %s.%s\n", Green, alias, d.Type, Reset)
			return
		}
	}
	fmt.Printf("%sError: Alias '%s' no encontrado en la configuración.%s\n", Red, alias, Reset)
	os.Exit(1)
}

func listCurrent() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("%sError al obtener home: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	configFile := filepath.Join(homeDir, "switchjdk", "downloads.json")
	var config Config
	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("%sError al leer configuración: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	json.Unmarshal(data, &config)
	if config.Current == nil {
		config.Current = make(map[string]string)
	}

	fmt.Println("🔎 Instalaciones disponibles:")
	for _, d := range config.Downloads {
		marker := ""
		if current, ok := config.Current[d.Type]; ok && current == d.Alias {
			marker = " (actual)"
		}
		fmt.Printf("- %s (tipo: %s)%s\n", d.Alias, d.Type, marker)
	}
}

func updatePath(binPath string) {
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf("$current = [Environment]::GetEnvironmentVariable('PATH', 'User'); $parts = $current -split ';'; $new = ($parts -join ';'); if ($new -notlike ('*' + '%s' + '*')) { $new += (';' + '%s') }; [Environment]::SetEnvironmentVariable('PATH', $new, 'User')", binPath, binPath))
	err := cmd.Run()
	if err != nil {
		fmt.Printf("%sError al actualizar PATH: %v%s\n", Red, err, Reset)
	} else {
		fmt.Printf("%sPATH actualizado con %s%s\n", Green, binPath, Reset)
	}
	// Recargar PATH en la sesión actual
	cmd = exec.Command("powershell", "-Command", "$machine = [Environment]::GetEnvironmentVariable('PATH', 'Machine'); $user = [Environment]::GetEnvironmentVariable('PATH', 'User'); $machine + ';' + $user")
	output, err := cmd.Output()
	if err == nil {
		fullPath := strings.TrimSpace(string(output))
		os.Setenv("PATH", fullPath)
	}
}

func setEnvVars(tipo, actualHome, alias string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("%sError al obtener home: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	configFile := filepath.Join(homeDir, "switchjdk", "downloads.json")
	var config Config
	if data, err := os.ReadFile(configFile); err == nil {
		json.Unmarshal(data, &config)
	}
	if config.Current == nil {
		config.Current = make(map[string]string)
	}
	if previous, ok := config.Current[tipo]; ok && previous != "" && previous != alias {
		// intentar remover bin previo del PATH
		targetDir := filepath.Join(homeDir, "switchjdk", previous)
		entries, err := os.ReadDir(targetDir)
		if err == nil {
			var subDir string
			for _, entry := range entries {
				if entry.IsDir() {
					subDir = entry.Name()
					break
				}
			}
			previousActualHome := targetDir
			if subDir != "" {
				previousActualHome = filepath.Join(targetDir, subDir)
			}

			// Intentar remover tanto home como home\bin
			removePathFromEnv(previousActualHome)
			removePathFromEnv(filepath.Join(previousActualHome, "bin"))
		}
	}
	config.Current[tipo] = alias
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configFile, data, 0644)

	switch tipo {
	case "java":
		setPersistentEnvVar("JAVA_HOME", actualHome)
		setPersistentEnvVar("JDK_HOME", actualHome)
		updatePath(filepath.Join(actualHome, "bin"))
	case "maven":
		setPersistentEnvVar("MAVEN_HOME", actualHome)
		updatePath(filepath.Join(actualHome, "bin"))
	case "gradle":
		setPersistentEnvVar("GRADLE_HOME", actualHome)
		updatePath(filepath.Join(actualHome, "bin"))
	case "php":
		setPersistentEnvVar("PHP_HOME", actualHome)
		// PHP en Windows suele tener el exe en la raíz, pero revisamos si hay bin
		if _, err := os.Stat(filepath.Join(actualHome, "bin")); err == nil {
			updatePath(filepath.Join(actualHome, "bin"))
		}
		updatePath(actualHome)
	case "go":
		setPersistentEnvVar("GOROOT", actualHome)
		updatePath(filepath.Join(actualHome, "bin"))
	case "node":
		setPersistentEnvVar("NODE_HOME", actualHome)
		// Node tiene el exe en la raíz en Windows (.zip)
		updatePath(actualHome)
	}

	fmt.Println("\n" + Yellow + "IMPORTANTE:" + Reset + " Para que los cambios surtan efecto en la terminal actual:")
	fmt.Println("1. Cierra y abre la terminal.")
	fmt.Println("2. O ejecuta: " + Cyan + "refreshenv" + Reset + " (si tienes Chocolatey).")
}

func setPersistentEnvVar(name, value string) {
	cmd := exec.Command("setx", name, value)
	if err := cmd.Run(); err != nil {
		fmt.Printf("%sError al setear %s: %v%s\n", Red, name, err, Reset)
	} else {
		fmt.Printf("%s%s establecido en %s%s\n", Green, name, value, Reset)
	}
	os.Setenv(name, value)
}

func removePathFromEnv(pathToRemove string) {
	if pathToRemove == "" {
		return
	}
	// Normalizar para evitar fallos por slash/backslash
	pathToRemove = strings.ReplaceAll(pathToRemove, "/", "\\")
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf("$current = [Environment]::GetEnvironmentVariable('PATH', 'User'); $parts = $current -split ';'; $filtered = $parts | Where-Object { $_ -ne '%s' -and $_ -ne '%s' }; $new = ($filtered -join ';'); [Environment]::SetEnvironmentVariable('PATH', $new, 'User')", pathToRemove, strings.ReplaceAll(pathToRemove, "\\", "/")))
	cmd.Run()
}
