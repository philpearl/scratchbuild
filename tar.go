package scratchbuild

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// TarDirectory builds a directory into a tar file. At the moment it does not support
// subdirectories. dir is the name of the directory to copy into the tar file. The tar file
// is written into w.
func TarDirectory(dir string, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}
	defer d.Close()

	fis, err := d.Readdir(0)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, fi := range fis {
		if fi.IsDir() {
			return fmt.Errorf("directories (%s) are not currently supported", fi.Name())
		}
		h, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return fmt.Errorf("failed building tar header for %s: %w", fi.Name(), err)
		}
		h.Name = filepath.Join(".", h.Name)

		if err := tw.WriteHeader(h); err != nil {
			return fmt.Errorf("failed writing header for %s: %w", fi.Name(), err)
		}

		f, err := os.Open(filepath.Join(dir, fi.Name()))
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			return fmt.Errorf("failed copying %s into tar file: %w", fi.Name(), err)
		}
		f.Close()
	}

	return nil
}
