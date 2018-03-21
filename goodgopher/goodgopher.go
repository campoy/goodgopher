package goodgopher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/src-d/go-github/github"
)

type server struct{}

// New returns an http.Handler that provides the entrypoints needed for goodgopher.
func New() (http.Handler, error) {
	return &server{}, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch kind := r.Header.Get("X-Github-Event"); kind {
	case "pull_request":
		var data github.PullRequestEvent
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			logrus.Errorf("could not decode request: %v", err)
			http.Error(w, "could not decode request", http.StatusInternalServerError)
			return
		}
		fmt.Println(*data.Action)
		commits := strings.Replace(*data.Repo.CommitsURL, "{/sha}", "", -1)
		fmt.Println(commits)
	default:
		http.Error(w, "unkown kind "+kind, http.StatusBadRequest)
	}
}
