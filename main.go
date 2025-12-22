package main

import (
	"github.com/kkk777-7/ingress2eg/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	cmd.Execute()
}
