package scratchbuild_test

import (
	"bytes"
	"log"

	"github.com/philpearl/scratchbuild"
)

func ExampleClient() {
	o := scratchbuild.Options{
		Dir:      "./testdata",
		Name:     "philpearl/test",
		BaseURL:  "https://index.docker.io",
		Tag:      "latest",
		User:     "philpearl",
		Password: "sekret",
	}

	b := &bytes.Buffer{}
	if err := scratchbuild.TarDirectory("./testdata", b); err != nil {
		log.Fatalf("failed to tar layer. %s", err)
	}

	c := scratchbuild.New(&o)

	token, err := c.Auth()
	if err != nil {
		log.Fatalf("failed to authorize. %s", err)
	}
	c.Token = token

	if err := c.BuildImage(&scratchbuild.ImageConfig{
		Entrypoint: []string{"/app"},
	}, b.Bytes()); err != nil {
		log.Fatalf("failed to build and send image. %s", err)
	}
}
