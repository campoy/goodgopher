package main

import (
	"io/ioutil"
	"net/http"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/campoy/goodgopher/goodgopher"
)

func main() {
	var config struct {
		IP      string `default:"0.0.0.0"`
		Port    string `default:"8080"`
		KeyPath string `default:"key.pem"`
		AppID   int    `default:"10114"`
		Verbose bool
	}
	envconfig.MustProcess("", &config)

	addr := config.IP + ":" + config.Port
	logrus.Infof("listening on %s", addr)

	if config.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debugf("logging set to debug level")
	}

	key, err := ioutil.ReadFile(config.KeyPath)
	if err != nil {
		logrus.Fatal(err)
	}

	h, err := goodgopher.New(config.AppID, key)
	if err != nil {
		logrus.Fatalf("could not create handler: %v", err)
	}

	logrus.Fatal(http.ListenAndServe(addr, h))
}
