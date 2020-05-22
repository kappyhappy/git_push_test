package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	git "github.com/kappyhappy/go-git/v5"
	"github.com/kappyhappy/go-git/v5/config"
	"github.com/kappyhappy/go-git/v5/plumbing"
	"github.com/kappyhappy/go-git/v5/plumbing/object"
	githttp "github.com/kappyhappy/go-git/v5/plumbing/transport/http"
	"github.com/pkg/errors"

	"math/rand"

	"github.com/google/go-github/v30/github"
	"golang.org/x/oauth2"
)

const (
	personalAccessToken string = "ThisValueShouldBeReplaced"
	targetRepo          string = "git_push_test"
	owner               string = "kappyhappy"
)

var rs1Letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randString1(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = rs1Letters[rand.Intn(len(rs1Letters))]
	}
	return string(b)
}

func cloneManifestRepository() (string, error) {
	dir, err := ioutil.TempDir("/tmp", targetRepo)
	if err != nil {
		fmt.Println("Failed to create temp dir")
		fmt.Println(err)
		return "", err
	}

	url := "https://github.com/" + owner + "/" + targetRepo + ".git"
	if _, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: "refs/heads/master",
		Depth:         1, // Shallow Clone
		Auth: &githttp.BasicAuth{
			Username: "dummy",
			Password: personalAccessToken,
		},
		Progress: nil,
	}); err != nil {
		fmt.Println("Failed to clone repository")
		fmt.Println(err)
		return "", err
	}

	return dir, nil
}

func commitChanges(ctx context.Context, baseDir string, client *github.Client) (*git.Repository, error) {
	randomStr := randString1(3)
	fileName := baseDir + "/random-string-" + randomStr + ".txt"
	head := "git-push-test"
	fmt.Println("File name is random-string-" + randomStr + ".txt")

	repo, err := git.PlainOpen(baseDir)
	if err != nil {
		fmt.Println("Failed to open repository")
		fmt.Println(err)
		return nil, errors.Wrapf(err, "Failed to open repository at %s", baseDir)
	}

	w, err := repo.Worktree()
	if err != nil {
		fmt.Println("Failed to open worktree")
		fmt.Println(err)
		return nil, errors.Wrap(err, "Failed to open worktree")
	}

	d1 := []byte("hello\ngit\n")
	err = ioutil.WriteFile(fileName, d1, 0644)
	if err != nil {
		fmt.Println("Failed to create file")
		fmt.Println(err)
		return nil, errors.Wrap(err, "Failed to open worktree")
	}

	if err := w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(head),
		Create: true,
		Keep:   true,
	}); err != nil {
		fmt.Println("Failed to git checkout")
		fmt.Println(err)
		return nil, errors.Wrap(err, "Failed to git checkout")
	}

	if _, err := w.Add("."); err != nil {
		fmt.Println("Failed to git add")
		fmt.Println(err)
		return nil, errors.Wrap(err, "Failed to git add")
	}

	commitMessage := fmt.Sprintf("Update")
	if _, err := w.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "git-push-tester",
			Email: "git-push-tester@sample.co.jp",
			When:  time.Now(),
		},
	}); err != nil {
		fmt.Println("Failed to git commit")
		fmt.Println(err)
		return nil, errors.Wrap(err, "Failed to commit")
	}

	return repo, nil
}

func pushChanges(ctx context.Context, client *github.Client, repo *git.Repository) (*github.PullRequest, error) {
	if err := repo.Push(&git.PushOptions{
		Prune: false,
		// Force: true,
		RefSpecs: []config.RefSpec{
			"refs/heads/master:refs/heads/qa/master",
			"refs/heads/git-push-test:refs/heads/rel/master",
			"refs/heads/*:refs/heads/*",
		},
		Auth: &githttp.BasicAuth{
			Username: "dummy",
			Password: personalAccessToken,
		},
	}); err != nil {
		fmt.Println("Failed to git push")
		fmt.Println(err)
		return nil, errors.Wrap(err, "Failed to push")
	}

	return nil, nil
}

func main() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: personalAccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	repoDir, err := cloneManifestRepository()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Git clone completed")

	repo, err := commitChanges(ctx, repoDir, client)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Git commit completed")

	_, err = pushChanges(ctx, client, repo)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Git push completed")
}
