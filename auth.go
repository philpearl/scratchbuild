package scratchbuild

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Auth gets a bearer token from the repository using the user and password from
// the client Options. If authentication is needed for your repository, call Auth
// before calling BuildImage
func (c *Client) Auth() (string, error) {
	// First do an empty get to get the auth challenge
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/v2/", nil)
	if err != nil {
		return "", err
	}
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed sending auth request: %w", err)
	}
	defer rsp.Body.Close()
	io.Copy(io.Discard, rsp.Body)

	if rsp.StatusCode == http.StatusOK {
		// no auth needed
		return "", nil
	}

	if rsp.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("unexpected status %s", rsp.Status)
	}

	// The Www-Authenticate header tells us where to go to get a token
	vals, err := parseWWWAuthenticate(rsp.Header.Get("Www-Authenticate"))
	if err != nil {
		return "", err
	}

	u, err := url.Parse(vals["realm"])
	if err != nil {
		return "", fmt.Errorf("could not parse authentication realm: %w", err)
	}
	q := u.Query()
	q.Set("service", vals["service"])
	q.Set("scope", "repository:"+c.Name+":pull,push")
	u.RawQuery = q.Encode()

	fmt.Printf("get %s\n", u)

	req, err = http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(c.User, c.Password)

	rsp, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed sending auth request: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s", rsp.Status)
	}
	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read auth response body: %w", err)
	}

	type token struct {
		Token string `json:"token"`
	}
	var tok token
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return tok.Token, nil
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
			return nil, fmt.Errorf("cannot parse Www-Authenticate header %s", raw)
		}
		v := kv[1]
		vals[kv[0]] = v[1 : len(v)-1]
	}
	return vals, nil
}
