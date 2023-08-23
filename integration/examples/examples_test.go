package examples

import (
	"os"
	"testing"

	"github.com/plamorg/voltproxy/config"
	"github.com/plamorg/voltproxy/dockerapi"
)

func TestExamples(t *testing.T) {
	examples := []string{
		"./middlewares/auth-forward.yml",
		"./middlewares/ip-allow.yml",
		"./additional-configuration.yml",
		"./basic.yml",
		"./health-check.yml",
		"./load-balancer.yml",
		"./multiple-middlewares.yml",
	}

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			confContent, err := os.ReadFile(example)
			if err != nil {
				t.Fatal(err)
			}

			conf, err := config.New(confContent)
			if err != nil {
				t.Fatal(err)
			}

			docker := dockerapi.NewMock()
			_, err = conf.Services(docker)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
