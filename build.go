package scratchbuild

import (
	"bytes"
	"compress/gzip"
	"time"
	// We need to import this to register the hash function for the digest
	_ "crypto/sha256"
	"encoding/json"

	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

// BuildImage builds a simple container image from a single layer and uploads it
// to a repository
func (c *Client) BuildImage(imageConfig *ImageConfig, layer []byte) error {

	dig := digest.FromBytes(layer)

	b := &bytes.Buffer{}
	gw := gzip.NewWriter(b)
	if _, err := gw.Write(layer); err != nil {
		return errors.Wrap(err, "failed to compress image layer")
	}
	if err := gw.Close(); err != nil {
		return errors.Wrap(err, "failed to compress image layer")
	}

	compressedLayer := b.Bytes()
	compressedDig := digest.FromBytes(compressedLayer)

	if err := c.sendBlob(compressedDig, compressedLayer); err != nil {
		return errors.Wrap(err, "failed to send image layer")
	}

	now := time.Now().UTC()
	image := Image{
		Created:      &now,
		Architecture: "amd64",
		OS:           "linux",
		Config:       *imageConfig,
		RootFS: RootFS{
			Type: "layers",
			// These must be the digest over the uncompressed content
			DiffIDs: []digest.Digest{
				dig,
			},
		},
	}

	imageData, err := json.Marshal(&image)
	if err != nil {
		return errors.Wrap(err, "could not marshal image config")
	}

	imageDigest := digest.FromBytes(imageData)

	// Perhaps we send the image config as a blob?
	if err := c.sendBlob(imageDigest, imageData); err != nil {
		return errors.Wrap(err, "could not send image description")
	}

	// Then a manifest to say what layers we have
	manifest := Manifest{
		Versioned: SchemaVersion,
		Layers: []Descriptor{
			{
				MediaType: MediaTypeLayer,
				Digest:    compressedDig,
				Size:      int64(len(compressedLayer)),
			},
		},
		Config: Descriptor{
			MediaType: MediaTypeImageConfig,
			Digest:    imageDigest,
			Size:      int64(len(imageData)),
		},
	}

	manifestData, err := json.Marshal(&manifest)
	if err != nil {
		return errors.Wrap(err, "could not marshal manifest")
	}

	manifestDigest := digest.FromBytes(manifestData)

	if err := c.sendManifest(manifestDigest, manifestData, MediaTypeManifest); err != nil {
		return errors.Wrap(err, "could not send manifest")
	}

	return nil
}
