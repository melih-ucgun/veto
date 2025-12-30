package resources

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Archive, uzak bir sunucudan sıkıştırılmış dosya (tar.gz, zip) indirip
// belirtilen hedef dizine açan kaynaktır.
type Archive struct {
	// YAML Fields
	Type        string `yaml:"type"`
	URL         string `yaml:"url"`
	Destination string `yaml:"destination"`
	Checksum    string `yaml:"checksum,omitempty"` // Format: "sha256:hash_value"
	Format      string `yaml:"format,omitempty"`   // "tar.gz", "zip". Boşsa URL'den tahmin edilir.

	// Internal
	Status string `yaml:"-"`
}

func (a *Archive) ID() string {
	return a.Destination
}

// Check, hedef dizinin var olup olmadığını kontrol eder.
func (a *Archive) Check() (bool, error) {
	info, err := os.Stat(a.Destination)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("archive check failed: %w", err)
	}

	if !info.IsDir() {
		return false, fmt.Errorf("destination exists but is not a directory: %s", a.Destination)
	}

	return true, nil
}

// Diff, mevcut durum ile istenen durum arasındaki farkı raporlar.
func (a *Archive) Diff() ([]string, error) {
	exists, err := a.Check()
	if err != nil {
		return nil, err
	}

	if !exists {
		return []string{fmt.Sprintf("Download and extract %s -> %s", a.URL, a.Destination)}, nil
	}

	// Eğer klasör varsa şimdilik değişiklik yok kabul ediyoruz.
	// İleride checksum kontrolü buraya da eklenebilir.
	return nil, nil
}

// Apply, arşivi indirir ve açar.
func (a *Archive) Apply() error {
	fmt.Printf("Downloading archive: %s\n", a.URL)

	// 1. Dosyayı geçici dizine indir
	tempFile, err := os.CreateTemp("", "monarch-archive-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	resp, err := http.Get(a.URL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	// İndirirken aynı zamanda hash hesapla
	hasher := sha256.New()
	tee := io.TeeReader(resp.Body, hasher)

	if _, err := io.Copy(tempFile, tee); err != nil {
		return fmt.Errorf("failed to save archive: %w", err)
	}

	// 2. Checksum Kontrolü
	if a.Checksum != "" {
		calculated := fmt.Sprintf("sha256:%x", hasher.Sum(nil))
		if calculated != a.Checksum {
			return fmt.Errorf("checksum mismatch! expected: %s, got: %s", a.Checksum, calculated)
		}
		fmt.Println("Checksum verified successfully.")
	}

	// Dosya işaretçisini başa al
	tempFile.Seek(0, 0)

	// 3. Format Belirleme
	format := a.Format
	if format == "" {
		if strings.HasSuffix(a.URL, ".tar.gz") || strings.HasSuffix(a.URL, ".tgz") {
			format = "tar.gz"
		} else if strings.HasSuffix(a.URL, ".zip") {
			format = "zip"
		} else {
			return fmt.Errorf("unknown archive format for URL: %s", a.URL)
		}
	}

	// 4. Hedef Dizini Oluştur
	if err := os.MkdirAll(a.Destination, 0755); err != nil {
		return fmt.Errorf("failed to create destination dir: %w", err)
	}

	// 5. Arşivi Aç
	switch format {
	case "tar.gz":
		if err := extractTarGz(tempFile, a.Destination); err != nil {
			return err
		}
	case "zip":
		// Zip için dosya ismine ihtiyaç var, tempFile.Name() kullanacağız
		if err := extractZip(tempFile.Name(), a.Destination); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	fmt.Printf("Archive extracted to: %s\n", a.Destination)
	return nil
}

// extractTarGz, .tar.gz dosyalarını açar
func extractTarGz(r io.Reader, dest string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		// Zip Slip zafiyetine karşı kontrol
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Dosyanın dizini yoksa oluştur
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

// extractZip, .zip dosyalarını açar
func extractZip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Zip Slip kontrolü
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
