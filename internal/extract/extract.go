package extract

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ProgressFunc recibe el progreso actual. total puede ser 0 si es desconocido.
type ProgressFunc func(current, total int, name string)

// Extract descomprime src dentro de dst, soportando .zip, .tar.gz y .tgz.
// Aplica protección contra zip-slip y rechaza enlaces simbólicos/duros por seguridad.
func Extract(ctx context.Context, src, dst string, onProgress ProgressFunc) error {
	lower := strings.ToLower(src)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(ctx, src, dst, onProgress)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(ctx, src, dst, onProgress)
	default:
		return fmt.Errorf("formato de archivo no soportado: solo .zip, .tar.gz y .tgz")
	}
}

func extractZip(ctx context.Context, src, dst string, onProgress ProgressFunc) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("no se pudo abrir zip: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	total := len(r.File)
	for i, f := range r.File {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		path, err := validatePath(dst, f.Name)
		if err != nil {
			return err
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("no se pudo crear archivo %s: %w", path, err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("no se pudo abrir entrada zip %s: %w", f.Name, err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("error al extraer %s: %w", f.Name, err)
		}

		if onProgress != nil {
			onProgress(i+1, total, f.Name)
		}
	}
	return nil
}

func extractTarGz(ctx context.Context, src, dst string, onProgress ProgressFunc) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("no se pudo abrir archivo: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("no se pudo abrir gzip: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	count := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error al leer tar: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		path, err := validatePath(dst, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			mode := os.FileMode(header.Mode) & os.ModePerm
			if mode == 0 {
				mode = 0755
			}
			if err := os.MkdirAll(path, mode); err != nil {
				return err
			}

		case tar.TypeReg:
			mode := os.FileMode(header.Mode) & os.ModePerm
			if mode == 0 {
				mode = 0644
			}
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
			if err != nil {
				return fmt.Errorf("no se pudo crear archivo %s: %w", path, err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("error al extraer %s: %w", header.Name, err)
			}
			outFile.Close()
			count++
			if onProgress != nil {
				onProgress(count, 0, header.Name)
			}

		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("enlaces no soportados por seguridad: %s", header.Name)

		default:
			// Ignorar otros tipos (fifos, dispositivos, etc.)
		}
	}
	return nil
}

// validatePath asegura que la ruta resultante quede dentro de base,
// evitando ataques de zip-slip y rutas absolutas.
func validatePath(base, name string) (string, error) {
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("ruta absoluta no permitida: %s", name)
	}

	joined := filepath.Join(base, name)
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("ruta fuera del destino: %s", name)
	}

	return absPath, nil
}
