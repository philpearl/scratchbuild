package scratchbuild

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

// Options contains configuration options for the client
type Options struct {
	Dir      string
	Name     string
	BaseURL  string
	User     string
	Password string
}

// Client lets you send a container up to a repository
type Client struct {
	Options
	Token string
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

	// First we create the upload. This tells us where to upload to
	u := strings.Join([]string{c.BaseURL, "v2", c.Name, "blobs/uploads/"}, "/")
	req, err := c.newRequest(http.MethodPost, u, nil)

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "blob upload failed")
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read body on blob upload response")
	}

	if rsp.StatusCode == http.StatusUnauthorized {
		fmt.Printf("headers: %#v\n", rsp.Header)
		authenticate := rsp.Header.Get("Www-Authenticate")
		parseWWWAuthenticate(authenticate)
	}

	if rsp.StatusCode != http.StatusAccepted {
		return errors.Errorf("unexpected status %s. %s", rsp.Status, string(body))
	}

	loc, err := rsp.Location()
	if err != nil {
		return errors.Wrap(err, "could not get location for blob upload")
	}

	q := loc.Query()
	q.Set("digest", digest.String())
	loc.RawQuery = q.Encode()

	log.Printf("posting data to %s\n", loc)

	r := bytes.NewReader(data)
	req, err = c.newRequest(http.MethodPut, loc.String(), r)
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", "application/octet-stream")

	rsp, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "blob upload failed")
	}
	defer rsp.Body.Close()
	body, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read body on blob upload response")
	}

	if rsp.StatusCode != http.StatusCreated {
		return errors.Errorf("unexpected status %s. %s", rsp.Status, string(body))
	}

	uuid := rsp.Header.Get("Docker-Upload-UUID")
	log.Printf("blob uuid is %s\n", uuid)

	return nil
}

func parseWWWAuthenticate(raw string) (map[string]string, error) {
	if !strings.HasPrefix(raw, "Bearer ") {
		return nil, errors.New("Www-Authenticate header does not start \"Bearer\"")
	}
	parts := strings.Split(raw[len("Bearer "):], ",")
	vals := make(map[string]string, len(parts))
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, errors.Errorf("cannot parse Www-Authenticate header %s", raw)
		}
		v := kv[1]
		vals[kv[0]] = v[1 : len(v)-1]
	}
	return vals, nil
}

func (c *Client) sendManifest(digest digest.Digest, data []byte, mediaType string) error {
	u := strings.Join([]string{c.BaseURL, "v2", c.Name, "manifests", "latest"}, "/")
	b := bytes.NewReader(data)
	req, err := c.newRequest(http.MethodPut, u, b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mediaType)

	log.Printf("Sending %s", u)

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
