package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

func main() {
	var config struct {
		IP   string `default:"0.0.0.0"`
		Port string `default:"8080"`
	}
	envconfig.MustProcess("", &config)

	http.HandleFunc("/", handler)
	addr := config.IP + ":" + config.Port
	logrus.Infof("listening on %s", addr)
	logrus.Fatal(http.ListenAndServe(addr, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello")
	b, _ := httputil.DumpRequest(r, true)
	logrus.Debugf("request: %s", b)
}
