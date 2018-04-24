package scratchbuild

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// TarDirectory builds a directory into a tar file. At the moment it does not support
// subdirectories
func TarDirectory(dir string, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	d, err := os.Open(dir)
	if err != nil {
		return errors.Wrap(err, "failed to open directory")
	}
	defer d.Close()

	fis, err := d.Readdir(0)
	if err != nil {
		return errors.Wrap(err, "failed to read directory")
	}

	for _, fi := range fis {
		if fi.IsDir() {
			return errors.Errorf("directories (%s) are not currently supported", fi.Name())
		}
		h, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return errors.Wrapf(err, "failed building tar header for %s", fi.Name())
		}
		h.Name = filepath.Join(".", h.Name)

		if err := tw.WriteHeader(h); err != nil {
			return errors.Wrapf(err, "failed writing header for %s", fi.Name())
		}

		f, err := os.Open(filepath.Join(dir, fi.Name()))
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			return errors.Wrapf(err, "failed copying %s into tar file", fi.Name())
		}
		f.Close()
	}

	return nil
}
