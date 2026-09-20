package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Minute

// ProgressFunc recibe bytes descargados y el total conocido.
// Si total es negativo o cero, el tamaño es desconocido.
type ProgressFunc func(downloaded, total int64)

// Fetch obtiene un archivo remoto o local y lo copia a dest.
// Si source comienza con http:// o https://, descarga; de lo contrario
// copia desde el sistema de archivos.
// Retorna una función de limpieza que debe ejecutarse cuando dest ya no se necesite.
func Fetch(ctx context.Context, source, dest string, onProgress ProgressFunc) (cleanup func(), err error) {
	cleanup = func() {}

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		if _, err := url.ParseRequestURI(source); err != nil {
			return cleanup, fmt.Errorf("URL inválida: %w", err)
		}
		if err := download(ctx, source, dest, onProgress); err != nil {
			return cleanup, err
		}
		cleanup = func() { _ = os.Remove(dest) }
		return cleanup, nil
	}

	absPath, err := filepath.Abs(source)
	if err != nil {
		return cleanup, fmt.Errorf("no se pudo resolver ruta local: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return cleanup, fmt.Errorf("archivo local no encontrado: %w", err)
	}
	if err := copyFile(absPath, dest, onProgress); err != nil {
		return cleanup, err
	}
	cleanup = func() { _ = os.Remove(dest) }
	return cleanup, nil
}

func download(ctx context.Context, urlStr, dest string, onProgress ProgressFunc) error {
	client := &http.Client{Timeout: defaultTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fallo la descarga: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("código de estado %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("no se pudo crear archivo temporal: %w", err)
	}
	defer out.Close()

	var r io.Reader = resp.Body
	if onProgress != nil {
		r = &progressReader{reader: resp.Body, total: resp.ContentLength, onProgress: onProgress}
	}
	if _, err := io.Copy(out, r); err != nil {
		return fmt.Errorf("error al escribir descarga: %w", err)
	}
	return nil
}

func copyFile(src, dst string, onProgress ProgressFunc) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	var r io.Reader = in
	if onProgress != nil {
		r = &progressReader{reader: in, total: info.Size(), onProgress: onProgress}
	}
	_, err = io.Copy(out, r)
	return err
}

type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	onProgress ProgressFunc
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.downloaded += int64(n)
	pr.onProgress(pr.downloaded, pr.total)
	return n, err
}
