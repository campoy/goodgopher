package goodgopher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"strings"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
)

type server struct {
	appID int
	key   []byte
}

// New returns an http.Handler that provides the entrypoints needed for goodgopher.
func New(appID int, key []byte) (http.Handler, error) {
	return &server{appID, key}, nil
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

		inst := data.GetInstallation()
		t, err := ghinstallation.New(http.DefaultTransport, s.appID, int(inst.GetID()), s.key)
		if err != nil {
			logrus.Error(err)
			http.Error(w, "could not authenticate with GitHub", http.StatusInternalServerError)
			return
		}
		client := github.NewClient(&http.Client{Transport: t})

		if err := processPullRequest(r.Context(), client, &data); err != nil {
			logrus.Error(err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "unkown kind "+kind, http.StatusBadRequest)
	}
}

func processPullRequest(ctx context.Context, client *github.Client, pr *github.PullRequestEvent) error {
	var (
		owner  = pr.Repo.GetOwner().GetLogin()
		repo   = pr.Repo.GetName()
		number = pr.GetNumber()
	)

	commits, _, err := client.PullRequests.ListCommits(ctx, owner, repo, pr.GetNumber(), nil)
	if err != nil {
		return fmt.Errorf("could not fetch commits: %v", err)
	}

	body := "hello 👋"
	path := "main.go"
	pos := 1
	comment := &github.PullRequestComment{
		Body:     &body,
		Path:     &path,
		CommitID: commits[len(commits)-1].SHA,
		Position: &pos,
	}
	_, _, err = client.PullRequests.CreateComment(ctx, owner, repo, number, comment)
	return err

	// commitsURL := strings.Replace(pr.PullRequest.GetCommitsURL(), "{/sha}", "", -1)
	// logrus.Debugf("fetching commits from %s", commitsURL)

	// var commits []*github.CommitResult
	// if err := httpGet(ctx, commitsURL, &commits); err != nil {
	// 	return err
	// }
	// for _, commit := range commits {
	// 	if err := processCommit(ctx, commit.Commit.GetURL()); err != nil {
	// 		return fmt.Errorf("could not process commit %d: %v", commit.SHA, err)
	// 	}
	// }
	return nil
}

func processCommit(ctx context.Context, url string) error {
	var commit struct{ Tree struct{ URL string } }
	if err := httpGet(ctx, url, &commit); err != nil {
		return err
	}

	return processTree(ctx, "/", commit.Tree.URL)
}

func processTree(ctx context.Context, path, url string) error {
	var tree github.Tree
	if err := httpGet(ctx, url, &tree); err != nil {
		return err
	}
	for _, entry := range tree.Entries {
		path := filepath.Join(path, entry.GetPath())

		switch entry.GetType() {
		case "blob":
			if !strings.HasSuffix(entry.GetPath(), ".go") {
				continue
			}
			logrus.Infof(entry.GetPath())
			content, err := base64.StdEncoding.DecodeString(entry.GetContent())
			if err != nil {
				return fmt.Errorf("could not decode %s: %v", path, err)
			}
			logrus.Infof("%s", content)
		case "tree":
			if err := processTree(ctx, path, entry.GetURL()); err != nil {
				return err
			}
		}
	}

	return nil
}

func httpGet(ctx context.Context, url string, dest interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if logrus.GetLevel() == logrus.DebugLevel {
		b, _ := httputil.DumpResponse(res, true)
		logrus.Debugf("commits response: %s\n", b)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("got %s", res.Status)
	}

	if err := json.NewDecoder(res.Body).Decode(dest); err != nil {
		return fmt.Errorf("could not decode: %v", err)
	}
	return nil
}
