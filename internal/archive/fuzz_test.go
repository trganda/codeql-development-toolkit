package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func seedTarGz(tb testing.TB) []byte {
	tb.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	files := []struct {
		name string
		body string
	}{
		{"hello.txt", "hello"},
		{"dir/nested.txt", "nested"},
	}
	for _, f := range files {
		hdr := &tar.Header{Name: f.name, Mode: 0600, Size: int64(len(f.body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			tb.Fatal(err)
		}
		if _, err := tw.Write([]byte(f.body)); err != nil {
			tb.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		tb.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		tb.Fatal(err)
	}
	return buf.Bytes()
}

func seedZip(tb testing.TB) []byte {
	tb.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("hello.txt")
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := w.Write([]byte("hello")); err != nil {
		tb.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		tb.Fatal(err)
	}
	return buf.Bytes()
}

// FuzzExtractTarGz feeds arbitrary bytes to ExtractTarGz. The contract is that
// it must never panic and must never write outside destDir, regardless of input.
func FuzzExtractTarGz(f *testing.F) {
	f.Add(seedTarGz(f))
	f.Add([]byte{})
	f.Add([]byte{0x1f, 0x8b}) // gzip magic only

	f.Fuzz(func(t *testing.T, data []byte) {
		tmp := t.TempDir()
		archivePath := filepath.Join(tmp, "in.tar.gz")
		destDir := filepath.Join(tmp, "out")
		if err := os.WriteFile(archivePath, data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(destDir, 0755); err != nil {
			t.Fatal(err)
		}
		_ = ExtractTarGz(archivePath, destDir)

		assertNoEscape(t, tmp, destDir)
	})
}

// FuzzExtractZip feeds arbitrary bytes to ExtractZip with the same contract.
func FuzzExtractZip(f *testing.F) {
	f.Add(seedZip(f))
	f.Add([]byte{})
	f.Add([]byte{'P', 'K', 0x03, 0x04}) // zip magic only

	f.Fuzz(func(t *testing.T, data []byte) {
		tmp := t.TempDir()
		archivePath := filepath.Join(tmp, "in.zip")
		destDir := filepath.Join(tmp, "out")
		if err := os.WriteFile(archivePath, data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(destDir, 0755); err != nil {
			t.Fatal(err)
		}
		_ = ExtractZip(archivePath, destDir)

		assertNoEscape(t, tmp, destDir)
	})
}

// assertNoEscape walks tmp and reports any regular file or symlink that was
// created outside destDir. This catches path-traversal regressions that the
// extractor's own checks were supposed to block.
func assertNoEscape(t *testing.T, tmp, destDir string) {
	t.Helper()
	cleanDest := filepath.Clean(destDir)
	_ = filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if path == tmp || path == cleanDest {
			return nil
		}
		// Anything under destDir is fine.
		if rel, rerr := filepath.Rel(cleanDest, path); rerr == nil && !filepath.IsAbs(rel) && rel != ".." && !startsWithDotDot(rel) {
			return nil
		}
		// The input archive itself lives in tmp; ignore it.
		base := filepath.Base(path)
		if base == "in.tar.gz" || base == "in.zip" || base == "out" {
			return nil
		}
		t.Errorf("entry written outside destDir: %s", path)
		return nil
	})
}

func startsWithDotDot(rel string) bool {
	return len(rel) >= 2 && rel[0] == '.' && rel[1] == '.'
}
