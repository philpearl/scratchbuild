package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

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

	var env multiString
	flag.Var(&env, "env", "Environment variables. Repeat to add more definitions, e.g. '-env PATH=/hat -env USER=postgras'")
	var volumes multiString
	flag.Var(&volumes, "vol", "Volumes. Repeat to add more definitions, e.g. '-vol /etc/myapp -env /var/myapp'")
	var entrypoint string
	flag.StringVar(&entrypoint, "entrypoint", "", "Entrypoint.")
	var labels multiPair
	flag.Var(&labels, "label", "Labels. Repeat to add more definitions, e.g. '-label label1=green -label label2=red'")

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
		Env: env,
	}

	if entrypoint != "" {
		imageConfig.Entrypoint = strings.Fields(entrypoint)
	}

	if len(labels) > 0 {
		imageConfig.Labels = make(map[string]string, len(labels))
		for _, l := range labels {
			imageConfig.Labels[l[0]] = l[1]
		}
	}

	if len(volumes) > 0 {
		imageConfig.Volumes = make(map[string]struct{}, len(volumes))
		for _, v := range volumes {
			imageConfig.Volumes[v] = struct{}{}
		}
	}

	if err := c.BuildImage(&imageConfig, b.Bytes()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build image. %s\n", err)
	}
}

type multiString []string

func (i *multiString) String() string {
	return "my string representation"
}

func (i *multiString) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type pair [2]string

type multiPair []pair

func (i *multiPair) String() string {
	return "lalala"
}

func (i *multiPair) Set(value string) error {

	p := strings.SplitN(value, "=", 2)
	if len(p) != 2 {
		return fmt.Errorf("should contain an =")
	}

	*i = append(*i, pair{p[0], p[1]})
	return nil
}
