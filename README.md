# switchtool - Gestor de Entornos para Windows 🚀

![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)
![Windows](https://img.shields.io/badge/Windows-10%2B-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)

**switchtool** es un CLI potente y minimalista diseñado para desarrolladores que necesitan cambiar rápidamente entre versiones de sus herramientas favoritas en Windows. Olvida configurar el %PATH% manualmente o pelear con variables de entorno cada vez que cambias de proyecto.

## 🌟 Lenguajes y Herramientas Soportadas

- **Java** (JDK)
- **Maven**
- **Gradle**
- **PHP**
- **Go** (GOROOT)
- **Node.js**

## ✨ Características

- **Descarga Automática**: Soporta URLs directas (Adoptium, Apache, etc.) o archivos locales.
- **Formatos Soportados**: Extrae automáticamente archivos `.zip`, `.tar.gz` y `.tgz`.
- **Gestión Inteligente de PATH**: Automatiza la limpieza y actualización del PATH de usuario mediante PowerShell.
- **Variables de Entorno**: Setea automáticamente `JAVA_HOME`, `MAVEN_HOME`, `GRADLE_HOME`, `PHP_HOME`, `GOROOT`, y `NODE_HOME`.
- **Detección de Directorios**: Si el archivo comprimido tiene una carpeta raíz extra (común en JDKs), **switchtool** la detecta y entra en ella automáticamente.
- **Interfaz Amigable**: Colores, barra de progreso y mensajes claros.

## 🛠 Instalación

### Opción 1: Descargar Binario
Ve a la sección de [Releases](https://github.com/Programmercito/switcher-version-tools/releases) y descarga el `switchtool.exe`. Agrégalo a tu PATH.

### Opción 2: Compilar desde fuente (requiere Go)
```bash
git clone https://github.com/Programmercito/switcher-version-tools.git
cd switcher-version-tools
go build -o switchtool.exe
```

## 🚀 Uso rápido

### 1. Instalar una nueva versión (Descarga)
```bash
# Sintaxis: switchtool <tipo> <alias> <url_o_path>
switchtool java jdk17 https://github.com/adoptium/temurin17-binaries/releases/download/jdk-17.0.9%2B9/OpenJDK17U-jdk_x64_windows_hotspot_17.0.9_9.zip
```

### 2. Usar un archivo local
```bash
switchtool php 8.2 C:\Downloads\php-8.2.10-Win32-vs16-x64.zip
```

### 3. Cambiar entre versiones
```bash
switchtool jdk17
switchtool php8.2
```

### 4. Listar lo que tienes instalado
```bash
switchtool list
# o
switchtool ls
```

## 📝 Notas de Windows
Para que los cambios en las variables de entorno se reflejen en la terminal actual:
1. Recomendamos usar **Chocolatey** y ejecutar `refreshenv`.
2. O simplemente cierra y abre tu terminal (PowerShell, CMD o Windows Terminal).

## 🤝 Contribuir
¡Toda ayuda es bienvenida! Si quieres añadir soporte para más herramientas o mejorar la lógica de extracción, abre un Issue o envía un Pull Request.

---
Hecho con amor en Go por [Programmercito](https://github.com/Programmercito).
