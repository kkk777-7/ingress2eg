package main

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/kkk777-7/ingress2eg/cmd"
)

func main() {
	cmd.Execute()
}
