package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

var (
	ErrMissingConfiguration  = errors.New("configuration file missing")
	ErrGitRepositoryNotFound = errors.New("fatal: not a git repository (or any of the parent directories): .git")
)

func ErrNotExist(path string) error {
	return errors.New(path + ": no such file or directory")
}

func ErrNotDirectory(path string) error {
	return errors.New(path + ": is not a directory")
}

// GitRepository represents the ".git" directory.
type GitRepository struct {
	WorkTree string    // WorkTree is the root directory where [GitRepository.GitDir] is located.
	GitDir   string    // GitDir is the ".git" directory under [GitRepository.WorkTree].
	Config   *ini.File // Config holds the parsed contents of the ".git/config" file.
}

// findGitDirectory searches for a ".git" directory starting from the provided startDir
// and traversing up to the root directory ("/"). If no directory is found, it returns
// an [ErrGitRepositoryNotFound].
func findGitDirectory(startDir string) (string, error) {
	dir := startDir
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return gitDir, nil
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break
		}

		dir = parentDir
	}

	return "", ErrGitRepositoryNotFound
}

// FromGitRepository creates a new [GitRepository]. It assumes that the workTree already contains a
// ".git" directory. It fails if there's no ".git/config" file or if the "core.repositoryformatversion"
// is not 0.
func FromGitRepository(workTree string) (*GitRepository, error) {
	gitDir, err := findGitDirectory(workTree)
	if err != nil {
		return nil, err
	}

	repo := &GitRepository{WorkTree: workTree, GitDir: gitDir}
	if !repo.HasFile([]string{"config"}) {
		return nil, ErrMissingConfiguration
	}

	repo.Config, err = ini.Load(repo.join("config"))
	if err != nil {
		return nil, err
	}

	// TODO:
	// core, err := repo.Config.GetSection("core")
	// key, err := core.GetKey("repositoryformatversion")

	return repo, nil
}

// NewGitRepository creates a new [GitRepository]. It's similar to [FromGitRepository] but assumes
// that the workTree does not contain a ".git" directory. The config file will be set to an empty INI,
// use [GitRepository.Config.SaveTo] to save changes.
func NewGitRepository(workTree string) (*GitRepository, error) {
	gitDir := filepath.Join(workTree, ".git")
	return &GitRepository{WorkTree: workTree, GitDir: gitDir, Config: ini.Empty()}, nil
}

// join joins the given path to [GitRepository.GitDir].
func (g *GitRepository) join(path ...string) string {
	return filepath.Join(append([]string{g.GitDir}, path...)...)
}

// HasFile reports whether the file specified by the filepath exists under [GitRepository.GitDir].
func (g *GitRepository) HasFile(filepath []string) bool {
	absPath := g.join(filepath...)
	file, _ := os.Stat(absPath)

	return file != nil && !file.IsDir()
}

// HasDir reports whether the give path is a directory under [GitRepository.GitDir]
func (g *GitRepository) HasDir(path ...string) bool {
	absPath := g.join(path...)

	file, err := os.Stat(absPath)
	switch {
	case os.IsNotExist(err):
		return false
	case !file.IsDir():
		return false
	default:
		return true
	}
}

// HasOrMkDir is similar to [GitRepository.HasDir] but creates the directory if it does ot exists. The created
// dir will have 0777 as permissions.
func (g *GitRepository) HasOrMkDir(path ...string) (bool, error) {
	if ok := g.HasDir(path...); !ok {
		if err := os.MkdirAll(g.join(path...), 0777); err != nil {
			return false, err
		}
	}

	return true, nil
}

// HasOrMkDirs is similar to [GitRepository.HasOrMkDir] but checks a list of paths.
// It returns true if all paths exist, otherwise returns false along with the index of the first path where the error occurs.
func (g *GitRepository) HasOrMkDirs(paths ...[]string) (int, error) {
	for i, p := range paths {
		if ok := g.HasDir(p...); !ok {
			if err := os.MkdirAll(g.join(p...), 0777); err != nil {
				return i, err
			}
		}
	}

	return 0, nil
}

// WriteFile writes the given content to the given file. The file will be joined to [GitRepository.GitDir].
func (g *GitRepository) WriteFile(file, content string) error {
	f, err := os.OpenFile(g.join(file), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(content))
	if err != nil {
		return err
	}

	return nil
}

type Git struct {
	repo *GitRepository
}

// Init initializes a new git repository. It creates the path if it does not
// exists. It fails if the path already has an git directory (.dir) and it is
// not empty or a file. Otherwise, it creates one.
func (g *Git) Init(path string) error {
	path, _ = filepath.Abs(path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0777); err != nil {
			return err
		}
	}

	repo, err := NewGitRepository(path)
	if err != nil {
		return err
	}

	if _, err := repo.HasOrMkDirs([]string{"branches"}, []string{"objects"}, []string{"refs", "tags"}, []string{"refs", "heads"}); err != nil {
		return err
	}

	if err := repo.WriteFile("description", "Unnamed repository; edit this file 'description' to name the repository.\n"); err != nil {
		return err
	}

	if err := repo.WriteFile("HEAD", "ref: refs/heads/master\n"); err != nil {
		return err
	}

	repo.Config.Section("core").Key("repositoryformatversion").SetValue("0")
	repo.Config.Section("core").Key("filemode").SetValue("false") // Disable permissions track
	repo.Config.Section("core").Key("bare").SetValue("false")
	repo.Config.SaveTo(repo.join("config"))

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("snap expects at least one command.")
		os.Exit(1)
	}

	git := &Git{}

	switch os.Args[1] {
	case "add":
	case "cat-file":
	case "check-ignore":
	case "checkout":
	case "commit":
	case "hash-object":
	case "init":
		path := ""
		if len(os.Args) >= 3 && os.Args[2] != "" {
			path = os.Args[2]
		} else {
			path = "."
		}

		if err := git.Init(path); err != nil {
			panic(err)
		}
	case "log":
	case "ls-files":
	case "ls-tree":
	case "rev-parse":
	case "rm":
	case "show-ref":
	case "status":
	case "tag":
	default:
		fmt.Println("Bad command")
	}
}
