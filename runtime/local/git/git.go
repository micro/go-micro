package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

type Gitter interface {
	Clone(repo string) error
	FetchAll(repo string) error
	Checkout(repo, branchOrCommit string) error
	RepoDir(repo string) string
}

type libGitter struct {
	folder string
}

func (g libGitter) Clone(repo string) error {
	fold := filepath.Join(g.folder, dirifyRepo(repo))
	exists, err := pathExists(fold)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = git.PlainClone(fold, false, &git.CloneOptions{
		URL:      repo,
		Progress: os.Stdout,
	})
	return err
}

func (g libGitter) FetchAll(repo string) error {
	repos, err := git.PlainOpen(filepath.Join(g.folder, dirifyRepo(repo)))
	if err != nil {
		return err
	}
	remotes, err := repos.Remotes()
	if err != nil {
		return err
	}

	err = remotes[0].Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
		Progress: os.Stdout,
		Depth:    1,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}

func (g libGitter) Checkout(repo, branchOrCommit string) error {
	if branchOrCommit == "latest" {
		branchOrCommit = "master"
	}
	repos, err := git.PlainOpen(filepath.Join(g.folder, dirifyRepo(repo)))
	if err != nil {
		return err
	}
	worktree, err := repos.Worktree()
	if err != nil {
		return err
	}

	if plumbing.IsHash(branchOrCommit) {
		return worktree.Checkout(&git.CheckoutOptions{
			Hash:  plumbing.NewHash(branchOrCommit),
			Force: true,
		})
	}

	return worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchOrCommit),
		Force:  true,
	})
}

func (g libGitter) RepoDir(repo string) string {
	return filepath.Join(g.folder, dirifyRepo(repo))
}

type binaryGitter struct {
	folder string
}

func (g binaryGitter) Clone(repo string) error {
	fold := filepath.Join(g.folder, dirifyRepo(repo), ".git")
	exists, err := pathExists(fold)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	fold = filepath.Join(g.folder, dirifyRepo(repo))
	cmd := exec.Command("git", "clone", repo, ".")

	err = os.MkdirAll(fold, 0777)
	if err != nil {
		return err
	}
	cmd.Dir = fold
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	return err
}

func (g binaryGitter) FetchAll(repo string) error {
	cmd := exec.Command("git", "fetch", "--all")
	cmd.Dir = filepath.Join(g.folder, dirifyRepo(repo))
	outp, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(outp))
	}
	return err
}

func (g binaryGitter) Checkout(repo, branchOrCommit string) error {
	if branchOrCommit == "latest" {
		branchOrCommit = "master"
	}
	cmd := exec.Command("git", "checkout", "-f", branchOrCommit)
	cmd.Dir = filepath.Join(g.folder, dirifyRepo(repo))
	outp, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(outp))
	}
	return nil
}

func (g binaryGitter) RepoDir(repo string) string {
	return filepath.Join(g.folder, dirifyRepo(repo))
}

func NewGitter(folder string) Gitter {
	if commandExists("git") {
		return binaryGitter{folder}
	}
	return libGitter{folder}
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func dirifyRepo(s string) string {
	s = strings.ReplaceAll(s, "https://", "")
	s = strings.ReplaceAll(s, "/", "-")
	return s
}

// exists returns whether the given file or directory exists
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// GetRepoRoot determines the repo root from a full path.
// Returns empty string and no error if not found
func GetRepoRoot(fullPath string) (string, error) {
	// traverse parent directories
	prev := fullPath
	for {
		current := prev
		exists, err := pathExists(filepath.Join(current, ".git"))
		if err != nil {
			return "", err
		}
		if exists {
			return current, nil
		}
		prev = filepath.Dir(current)
		// reached top level, see:
		// https://play.golang.org/p/rDgVdk3suzb
		if current == prev {
			break
		}
	}
	return "", nil
}

const defaultRepo = "github.com/micro/services"

// Source is not just git related @todo move
type Source struct {
	// is it a local folder intended for a local runtime?
	Local bool
	// absolute path to service folder in local mode
	FullPath string
	// path of folder to repo root
	// be it local or github repo
	Folder string
	// github ref
	Ref string
	// for cloning purposes
	// blank for local
	Repo string
	// dir to repo root
	// blank for non local
	LocalRepoRoot string
}

// Name to be passed to RPC call runtime.Create Update Delete
// eg: `helloworld/api`, `crufter/myrepo/helloworld/api`, `localfolder`
func (s *Source) RuntimeName() string {
	if s.Repo == "github.com/micro/services" || s.Repo == "" {
		return s.Folder
	}
	return fmt.Sprintf("%v/%v", strings.ReplaceAll(s.Repo, "github.com/", ""), s.Folder)
}

// Source to be passed to RPC call runtime.Create Update Delete
// eg: `helloworld`, `github.com/crufter/myrepo/helloworld`, `/path/to/localrepo/localfolder`
func (s *Source) RuntimeSource() string {
	if s.Local {
		return s.FullPath
	}
	if s.Repo == "github.com/micro/services" || s.Repo == "" {
		return s.Folder
	}
	return fmt.Sprintf("%v/%v", s.Repo, s.Folder)
}

// ParseSource parses a `micro run/update/kill` source.
func ParseSource(source string) (*Source, error) {
	// If github is not present, we got a shorthand for `micro/services`
	if !strings.Contains(source, "github.com") {
		source = "github.com/micro/services/" + source
	}
	if !strings.Contains(source, "@") {
		source += "@latest"
	}
	ret := &Source{}
	refs := strings.Split(source, "@")
	ret.Ref = refs[1]
	parts := strings.Split(refs[0], "/")
	ret.Repo = strings.Join(parts[0:3], "/")
	if len(parts) > 1 {
		ret.Folder = strings.Join(parts[3:], "/")
	}

	return ret, nil
}

// ParseSourceLocal detects and handles local pathes too
// workdir should be used only from the CLI @todo better interface for this function.
// PathExistsFunc exists only for testing purposes, to make the function side effect free.
func ParseSourceLocal(workDir, source string, pathExistsFunc ...func(path string) (bool, error)) (*Source, error) {
	var pexists func(string) (bool, error)
	if len(pathExistsFunc) == 0 {
		pexists = pathExists
	} else {
		pexists = pathExistsFunc[0]
	}
	var localFullPath string
	if len(workDir) > 0 {
		localFullPath = filepath.Join(workDir, source)
	} else {
		localFullPath = source
	}
	if exists, err := pexists(localFullPath); err == nil && exists {
		localRepoRoot, err := GetRepoRoot(localFullPath)
		if err != nil {
			return nil, err
		}
		var folder string
		// If the local repo root is a top level folder, we are not in a git repo.
		// In this case, we should take the last folder as folder name.
		if localRepoRoot == "" {
			folder = filepath.Base(localFullPath)
		} else {
			folder = strings.ReplaceAll(localFullPath, localRepoRoot+string(filepath.Separator), "")
		}

		return &Source{
			Local:         true,
			Folder:        folder,
			FullPath:      localFullPath,
			LocalRepoRoot: localRepoRoot,
			Ref:           "latest", // @todo consider extracting branch from git here
		}, nil
	}
	return ParseSource(source)
}

// CheckoutSource for the local runtime server
// folder is the folder to check out the source code to
// Modifies source path to set it to checked out repo absolute path locally.
func CheckoutSource(folder string, source *Source) error {
	// if it's a local folder, do nothing
	if exists, err := pathExists(source.FullPath); err == nil && exists {
		return nil
	}
	gitter := NewGitter(folder)
	repo := source.Repo
	if !strings.Contains(repo, "https://") {
		repo = "https://" + repo
	}
	// Always clone, it's idempotent and only clones if needed
	err := gitter.Clone(repo)
	if err != nil {
		return err
	}
	source.FullPath = filepath.Join(gitter.RepoDir(source.Repo), source.Folder)
	return gitter.Checkout(repo, source.Ref)
}

// code below is not used yet

var nameExtractRegexp = regexp.MustCompile(`((micro|web)\.Name\(")(.*)("\))`)

func extractServiceName(fileContent []byte) string {
	hits := nameExtractRegexp.FindAll(fileContent, 1)
	if len(hits) == 0 {
		return ""
	}
	hit := string(hits[0])
	return strings.Split(hit, "\"")[1]
}
