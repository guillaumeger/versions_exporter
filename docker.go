package main

import (
	"github.com/docker/distribution/registry/client"
	"net/http"
)

func main() {
	reg, err := client.NewRegistry("https://registry-1.docker.io", http.)
}
