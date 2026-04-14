package bundle

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// copyPackTo copies a pack's on-disk directory under destRoot/<scope>/<name>/<version>
// and returns a new Pack pointing at the copy.
func copyPackTo(p *Pack, destRoot string) (*Pack, error) {
	scope := p.Config.Scope()
	name := p.Config.PackName()
	version := p.Config.Version
	if version == "" {
		version = "0.0.0"
	}
	destDir := filepath.Join(destRoot, scope, name, version)
	slog.Debug("Copying pack", "from", p.Dir(), "to", destDir)
	if err := copyDir(p.Dir(), destDir); err != nil {
		return nil, fmt.Errorf("copying pack %s: %w", p.Config.Name, err)
	}
	return &Pack{
		YmlPath: filepath.Join(destDir, filepath.Base(p.YmlPath)),
		Config:  p.Config,
		Kind:    p.Kind,
		Deps:    p.Deps,
	}, nil
}

// copyDir recursively copies src to dst, preserving symlinks.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		linfo, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if linfo.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
