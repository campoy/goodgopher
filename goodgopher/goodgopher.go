package goodgopher

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
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
	kind := r.Header.Get("X-Github-Event")
	logrus.Debugf("received hook of kind %s", kind)
	switch kind {
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
		// default:
		// 	http.Error(w, "unkown kind "+kind, http.StatusBadRequest)
	}
}

type comment struct {
	path    string
	line    int
	message string
}

func processRepo(repo, ref string) ([]comment, error) {
	logrus.Infof("processing %s:%s", repo, ref)
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, fmt.Errorf("could not create temp dir: %v", err)
	}
	// defer os.RemoveAll(dir)

	importPath := strings.TrimSuffix(strings.TrimPrefix(repo, "git://"), ".git")

	r := runner{gopath: dir, path: importPath}
	absPath := filepath.Join(dir, "src", importPath)
	os.MkdirAll(absPath, os.ModePerm)

	opts := &git.CloneOptions{
		URL:           repo,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", ref)),
		SingleBranch:  true,
	}
	logrus.Infof("git clone %s -b %s", opts.URL, opts.ReferenceName)
	_, err = git.PlainClone(absPath, false, opts)
	if err != nil {
		return nil, fmt.Errorf("could not clone repo: %v", err)
	}

	logrus.Info("fetching dependencies")
	out, err := r.run("go", "get", ".")
	if err != nil {
		return nil, fmt.Errorf("could not fetch dependencies: %v\n%s", err, out)
	}

	logrus.Info("vetting code")
	out, err = r.run("megacheck", "./...")
	if err == nil {
		return nil, nil
	}

	var comments []comment

	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		ps := strings.SplitN(s.Text(), ":", 4)
		if len(ps) != 4 {
			logrus.Errorf("unparsable line %s", s.Text())
		}
		path, line, _, msg := ps[0], ps[1], ps[2], ps[3]
		path = strings.TrimPrefix(path, absPath+"/")
		lineNumber, err := strconv.Atoi(line)
		if err != nil {
			logrus.Errorf("bad line number: %v", err)
		}
		comments = append(comments, comment{path, lineNumber, msg})
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("could not parse output: %v", err)
	}
	return comments, nil
}

type runner struct{ gopath, path string }

func (r runner) run(bin string, args ...string) ([]byte, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = filepath.Join(r.gopath, "src", r.path)
	cmd.Env = append(os.Environ(), "GOPATH="+r.gopath)
	return cmd.CombinedOutput()
}

func processPullRequest(ctx context.Context, client *github.Client, pr *github.PullRequestEvent) error {
	var (
		owner    = pr.Repo.GetOwner().GetLogin()
		repoName = pr.Repo.GetName()
		number   = pr.GetNumber()
	)

	// fs := osfs.New(os.TempDir())
	// storage, err := filesystem.NewStorage(fs)
	// if err != nil {
	// 	return fmt.Errorf("could not create storage: %v", err)
	// }
	// repo, err := git.Clone(storage, fs, &git.CloneOptions{
	// 	URL: pr.Repo.GetURL(),
	// })
	// logrus.Infof("cloned %s", pr.Repo.GetURL())
	// if err != nil {
	// 	return fmt.Errorf("could not open repo: %v", err)
	// }
	// if err := repo.Fetch(&git.FetchOptions{
	// 	RefSpecs: []config.RefSpec{},
	// }); err != nil {
	// 	return fmt.Errorf("could not fetch: %v", err)
	// }

	// // git.PlainClone("/tmp/foo", true, &git.CloneOptions{URL: pr.Repo.GetURL()})
	// commits, _, err := client.PullRequests.ListCommits(ctx, owner, repoName, pr.GetNumber(), nil)
	// if err != nil {
	// 	return fmt.Errorf("could not fetch commits: %v", err)
	// }

	comments, err := processRepo(pr.PullRequest.Head.Repo.GetGitURL(), pr.PullRequest.Head.GetRef())
	if err != nil {
		return fmt.Errorf("could not process repo: %v", err)
	}

	for _, comment := range comments {
		comment := &github.PullRequestComment{
			Body:     &comment.message,
			Path:     &comment.path,
			CommitID: pr.PullRequest.Head.SHA,
			Position: &comment.line,
		}
		_, _, err = client.PullRequests.CreateComment(ctx, owner, repoName, number, comment)
		if err != nil {
			return fmt.Errorf("could not comment on repo: %v", err)
		}
	}
	return nil

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
	// return nil
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
