package main

import (
	"fmt"
	"github.com/aapelismith/frp-service-provider/pkg/client/frp"
	"os"
	"sigs.k8s.io/yaml"
)

func main() {
	opt := frp.Options{}

	data, err := os.ReadFile("./config/config.yaml")
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(data, &opt); err != nil {
		panic(err)
	}

	fmt.Printf("=====%+v", opt)
}
