# SwitchJDK - Gestor de Versiones Java, Maven y Gradle 🚀

![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)
![Windows](https://img.shields.io/badge/Windows-10%2B-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)

¿Cansado de lidiar con versiones de Java, Maven y Gradle en tu máquina Windows? ¡SwitchJDK es tu solución definitiva! Este CLI potente y fácil de usar te permite descargar, instalar y cambiar entre versiones con un solo comando. Olvídate de configuraciones manuales y PATH confusos. 💻✨

## 🌟 Características Destacadas

- **Descarga Automática**: Descarga JDK, Maven o Gradle desde URLs o archivos locales con barra de progreso en tiempo real.
- **Cambio Instantáneo**: Cambia entre versiones instaladas con un alias simple.
- **Gestión Inteligente de PATH**: Agrega rutas directas al PATH, eliminando versiones anteriores automáticamente.
- **Lista de Versiones**: Ve qué versiones tienes activas y sus rutas con un comando.
- **Configuración Persistente**: Guarda todo en un archivo JSON limpio y editable.
- **Colores y Emojis**: Interfaz amigable con colores y mensajes claros. 🎨
- **Seguro**: Verifica integridad de archivos ZIP y evita sobrescrituras accidentales.

## 📦 Instalación

### Prerrequisitos
- **Windows 10 o superior**
- **Go 1.21+** (para compilar desde fuente)
- **Chocolatey** (opcional, pero recomendado para `refreshenv`)

### Compilar desde Fuente
1. Clona el repositorio:
   ```bash
   git clone https://github.com/Programmercito/miapp-cli.git
   cd miapp-cli
   ```

2. Compila:
   ```bash
   go build -o switchjdk.exe
   ```

3. (Opcional) Agrega al PATH global para usarlo desde cualquier lugar.

### Instalar Chocolatey (para `refreshenv`)
Si no tienes Chocolatey, instálalo para recargar el entorno automáticamente:
```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
```

## 🚀 Uso

### Sintaxis Básica
```bash
switchjdk.exe <comando> [argumentos]
```

### Comandos Disponibles

#### 1. Descargar e Instalar una Versión
Descarga desde una URL y configura un alias.
```bash
switchjdk.exe <URL> <tipo> <alias>
```
- **URL**: Enlace de descarga (ej. de Adoptium, Apache Maven, Gradle).
- **Tipo**: `java`, `maven` o `gradle`.
- **Alias**: Nombre corto para la versión (ej. `javav11`, `mav3`).

**Ejemplo**:
```bash
switchjdk.exe https://github.com/adoptium/temurin17-binaries/releases/download/jdk-17.0.9%2B9/OpenJDK17U-jdk_x64_windows_hotspot_17.0.9_9.zip java javav17
```
Descarga JDK 17 y lo configura como `javav17`.

#### 2. Cambiar a una Versión Instalada
```bash
switchjdk.exe <alias>
```
Cambia al alias especificado, actualizando variables de entorno y PATH.

**Ejemplo**:
```bash
switchjdk.exe javav17
```
Activa Java 17. ¡El PATH se actualiza automáticamente!

#### 3. Listar Versiones Actuales
```bash
switchjdk.exe list
```
Muestra las versiones activas para cada tipo con sus rutas `bin`.

**Ejemplo de Salida**:
```
Aliases actuales:
java: javav17 -> C:\Users\TuUsuario\switchjdk\javav17\jdk-17.0.9+9\bin
maven: mav3 -> C:\Users\TuUsuario\switchjdk\mav3\apache-maven-3.9.5\bin
```

### Notas Importantes
- Después de cambiar versiones, ejecuta `refreshenv` en PowerShell para que los cambios surjan efecto en la sesión actual. (Viene con Chocolatey).
- Los archivos se extraen en `C:\Users\TuUsuario\switchjdk\<alias>`.
- No puedes tener un alias llamado `list` (reservado para el comando).

## 📋 Ejemplos Prácticos

### Instalar y Cambiar a Java 11
```bash
# Descargar
switchjdk.exe https://github.com/adoptium/temurin11-binaries/releases/download/jdk-11.0.21%2B9/OpenJDK11U-jdk_x64_windows_hotspot_11.0.21_9.zip java javav11

# Cambiar
switchjdk.exe javav11

# Verificar
java --version
refreshenv
```

### Instalar Maven 3.9
```bash
switchjdk.exe https://downloads.apache.org/maven/maven-3/3.9.5/binaries/apache-maven-3.9.5-bin.zip maven mav395
switchjdk.exe mav395
mvn --version
refreshenv
```

### Ver Versiones Activas
```bash
switchjdk.exe list
```

## 🛠️ Cómo Funciona Internamente

- **Descargas**: Usa Go's `net/http` con barra de progreso. Soporta ZIP y TAR.GZ.
- **Extracción Segura**: Verifica rutas para evitar ataques ZIP maliciosos.
- **Configuración**: Archivo `downloads.json` en `switchjdk/` con lista de descargas y aliases actuales.
- **PATH Management**: Agrega rutas directas, removiendo anteriores para evitar conflictos.
- **Variables de Entorno**: Setea `JAVA_HOME`, `MAVEN_HOME`, `GRADLE_HOME` con `setx`.

## 🤝 Contribuir

¡Las contribuciones son bienvenidas! 
- Reporta bugs o solicita features en Issues.
- Envía PRs con mejoras.
- Sigue el estilo de código Go estándar.

## 📄 Licencia

MIT License - ¡Úsalo libremente!

---

Hecho con ❤️ en Go. Simplifica tu desarrollo Java/Maven/Gradle. ¡Disfruta! 🎉