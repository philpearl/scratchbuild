package scratchbuild

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

// Options contains configuration options for the client
type Options struct {
	// Dir is the directory that we build the container from
	Dir string
	// Name is the name of the repository
	Name string
	// BaseURL is the base URL of the repository. For Docker this is https://index.docker.io
	// For GCR it is https://gcr.io
	BaseURL string
	//
	User     string
	Password string
	// Token is the bearer token for the repository. For GCR you can use $(gcloud auth print-access-token).
	// For Docker, supply your Docker Hub username and password instead.
	Token string
	// Tag is the tag for the image. Set to "latest" if you're out of ideas
	Tag string
}

// Client lets you send a container up to a repository
type Client struct {
	Options
}

// New creates a new Client
func New(o *Options) *Client {
	return &Client{
		Options: *o,
	}
}

func (c *Client) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	return req, nil
}

func (c *Client) sendBlob(digest digest.Digest, data []byte) error {

	uploaded, err := c.isBlobUploaded(digest)
	if err != nil {
		return errors.Wrap(err, "could not check if blob is already uploaded")
	}
	if uploaded {
		fmt.Printf("blob already uploaded\n")
		return nil
	}

	// The repository tells us where the blob should be uploaded to
	loc, err := c.getBlobUploadLocation()
	if err != nil {
		return errors.Wrap(err, "could not get location for blob upload")
	}

	if err := c.uploadBlob(loc, digest, data); err != nil {
		return errors.Wrap(err, "blob upload failed")
	}

	return nil
}

func (c *Client) isBlobUploaded(digest digest.Digest) (bool, error) {
	u := strings.Join([]string{c.BaseURL, "v2", c.Name, "blobs", digest.String()}, "/")

	req, err := c.newRequest(http.MethodHead, u, nil)
	if err != nil {
		return false, errors.Wrap(err, "could nto build request")
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "blob upload failed")
	}

	return rsp.StatusCode == http.StatusOK, nil
}

func (c *Client) getBlobUploadLocation() (*url.URL, error) {
	u := strings.Join([]string{c.BaseURL, "v2", c.Name, "blobs/uploads/"}, "/")
	req, err := c.newRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "blob upload failed")
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read body on blob upload response")
	}

	if rsp.StatusCode != http.StatusAccepted {
		return nil, errors.Errorf("unexpected status %s. %s", rsp.Status, string(body))
	}

	return rsp.Location()
}

func (c *Client) uploadBlob(loc *url.URL, digest digest.Digest, data []byte) error {
	q := loc.Query()
	q.Set("digest", digest.String())
	loc.RawQuery = q.Encode()

	r := bytes.NewReader(data)
	req, err := c.newRequest(http.MethodPut, loc.String(), r)
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", "application/octet-stream")

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "blob upload failed")
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read body on blob upload response")
	}

	if rsp.StatusCode != http.StatusCreated {
		return errors.Errorf("unexpected status %s. %s", rsp.Status, string(body))
	}

	return nil
}

func (c *Client) sendManifest(digest digest.Digest, data []byte, mediaType string) error {
	u := strings.Join([]string{c.BaseURL, "v2", c.Name, "manifests", c.Tag}, "/")
	b := bytes.NewReader(data)
	req, err := c.newRequest(http.MethodPut, u, b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mediaType)

	log.Printf("Sending %s", u)

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "manifest upload failed")
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read body on manifest upload response")
	}

	if rsp.StatusCode != http.StatusCreated && rsp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status %s. %s", rsp.Status, string(body))
	}

	return nil
}
