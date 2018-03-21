package main

import (
	"net/http"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/campoy/goodgopher/goodgopher"
)

func main() {
	var config struct {
		IP   string `default:"0.0.0.0"`
		Port string `default:"8080"`
	}
	envconfig.MustProcess("", &config)
	addr := config.IP + ":" + config.Port
	logrus.Infof("listening on %s", addr)

	h, err := goodgopher.New()
	if err != nil {
		logrus.Fatalf("could not create handler: %v", err)
	}

	logrus.Fatal(http.ListenAndServe(addr, h))
}
