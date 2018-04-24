package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/philpearl/scratchbuild"
)

func validate(o *scratchbuild.Options) error {
	if o.Name == "" {
		return fmt.Errorf("You must specify a name for the image")
	}
	return nil
}

func main() {
	var o scratchbuild.Options

	flag.StringVar(&o.Dir, "dir", "./", "Directory containing container content")
	flag.StringVar(&o.Name, "name", "", "Image name")
	// THe docker repository is https://index.docker.io
	flag.StringVar(&o.BaseURL, "regurl", "https://eu.gcr.io", "Registry URL")
	// If you don't have a token, pass in a user name and password and we'll go and
	// get one. For the docker repository this is your Docker Hub username & password.
	// Don't use these for the GCP repository
	flag.StringVar(&o.User, "user", "", "Registry user name")
	flag.StringVar(&o.Password, "password", "", "Registry password")
	flag.StringVar(&o.Token, "token", "", "Repository bearer token. For the GCP repository use this with $(gcloud auth print-access-token)")
	flag.StringVar(&o.Tag, "tag", "latest", "Image tag")

	flag.Parse()

	if err := validate(&o); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(1)
	}

	c := scratchbuild.New(&o)

	if c.Token == "" {
		var err error
		c.Token, err = c.Auth()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to authenticate. %s\n", err)
			os.Exit(1)
		}
	}

	b := &bytes.Buffer{}
	if err := scratchbuild.TarDirectory(c.Dir, b); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build tar file. %s\n", err)
		os.Exit(1)
	}

	imageConfig := scratchbuild.ImageConfig{
		Env: []string{"SSL_CERT_FILE=/ca-certificates.crt", "GOOGLE_APPLICATION_CREDENTIALS=/fakegcpcreds.json"},
		Labels: map[string]string{
			"org.label-schema.schema-version": "1.0",
			"org.label-schema.vendor":         "ravelin",
			"org.label-schema.vcs-url":        "https://github.com/unravelin/core",
			"org.label-schema.name":           "$NAME",
			"org.label-schema.build_date":     "$DATE",
			"org.label-schema.version":        "$VERSION",
			"org.label-schema.vcs-ref":        "$VCS_REF",
		},
		Volumes: map[string]struct{}{
			"/etc/ravelin": struct{}{},
		},
		Entrypoint: []string{"/app"},
	}

	if err := c.BuildImage(&imageConfig, b.Bytes()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build image. %s\n", err)
	}
}
