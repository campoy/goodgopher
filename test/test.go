package main

import (
	"fmt"
	"log"

	"github.com/sirupsen/logrus"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/diff"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func main() {
	if err := diffHeadAndPrevious("https://github.com/campoy/goodgopher", "pr"); err != nil {
		log.Fatal(err)
	}
}

func diffHeadAndPrevious(path, branch string) error {
	logrus.Infof("cloning")
	opts := &git.CloneOptions{
		URL:           path,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		SingleBranch:  true,
	}
	repo, err := git.PlainClone("repo", false, opts)
	if err != nil {
		return fmt.Errorf("could not clone repo: %v", err)
	}

	logrus.Infof("done cloning")
	headRef, err := repo.Head()
	if err != nil {
		return fmt.Errorf("could not fetch head: %v", err)
	}

	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return fmt.Errorf("could not get commit for head: %v", err)
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return fmt.Errorf("could not fetch head tree: %v", err)
	}

	if headCommit.NumParents() > 1 {
		return fmt.Errorf("not sure how to handle merge commits yet")
	} else if headCommit.NumParents() == 0 {
		return nil
	}
	prevCommit, err := headCommit.Parent(0)
	if err != nil {
		return fmt.Errorf("could not fetch previous commit: %v", err)
	}
	prevTree, err := prevCommit.Tree()
	if err != nil {
		return fmt.Errorf("could not fetch previous tree: %v", err)
	}

	changes, err := object.DiffTree(headTree, prevTree)
	if err != nil {
		return fmt.Errorf("could not diff trees: %v", err)
	}

	patch, err := changes.Patch()
	if err != nil {
		return fmt.Errorf("could not get change patch: %v", err)
	}
	for _, p := range patch.FilePatches() {
		if p.IsBinary() {
			continue
		}
		for l, chunk := range p.Chunks() {
			if chunk.Type() != diff.Add {
				continue
			}
			fmt.Printf("%3d %s", l, chunk.Content())
		}
	}

	return nil
}
