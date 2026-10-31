package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Minute
const maxAttempts = 3

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
			_ = os.Remove(dest)
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
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(time.Duration(attempt) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			if onProgress != nil {
				onProgress(0, 0)
			}
		}
		err = downloadOnce(ctx, client, urlStr, dest, onProgress)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err == nil || !retryable(err) {
			return err
		}
	}
	return fmt.Errorf("descarga fallida después de %d intentos: %w", maxAttempts, err)
}

type statusError int

func (e statusError) Error() string { return fmt.Sprintf("código de estado %d", e) }

func retryable(err error) bool {
	var status statusError
	if errors.As(err, &status) {
		return status == http.StatusTooManyRequests || status >= 500
	}
	var networkError net.Error
	return errors.As(err, &networkError) || errors.Is(err, io.ErrUnexpectedEOF)
}

func downloadOnce(ctx context.Context, client *http.Client, urlStr, dest string, onProgress ProgressFunc) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return err
	}

	// Sin compresión el servidor informa Content-Length y se puede mostrar la barra.
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fallo la descarga: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return statusError(resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("no se pudo crear archivo temporal: %w", err)
	}
	var r io.Reader = resp.Body
	if onProgress != nil {
		r = &progressReader{reader: resp.Body, total: responseSize(resp), onProgress: onProgress}
	}
	if _, err := io.Copy(out, r); err != nil {
		_ = out.Close()
		return fmt.Errorf("error al escribir descarga: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("error al cerrar descarga: %w", err)
	}
	return nil
}

// responseSize retorna el tamaño total, usando X-Identity-Content-Length
// (que envía Google) cuando Content-Length no está disponible.
func responseSize(resp *http.Response) int64 {
	if resp.ContentLength > 0 {
		return resp.ContentLength
	}
	if n, err := strconv.ParseInt(resp.Header.Get("X-Identity-Content-Length"), 10, 64); err == nil && n > 0 {
		return n
	}
	return resp.ContentLength
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
