package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func main() {
	if err := newCommand().Execute(); err != nil {
		os.Exit(2)
	}
}

func newCommand() *cobra.Command {
	root := cobra.Command{
		Use:  "link2code FILES...",
		RunE: runCommand,
		Args: cobra.ArbitraryArgs,
		Short: `
link2code crafts direct URLs to source on GitHub

For every file given, it compares local revisions to those in origin. The most recent,
common revision is used for the direct link. Line numbers, and ranges, are supported
by appending ":start[-end]" to the filepath.

Files in trees that are not git repositories are skipped.`,
		Example: `
link2code Makefile
link2code Makefile:5-10
link2code repo1/Makefile repo2/cmd/my-tool.go repo3/README.md:25-30

rg 'search term' -n | link2code
		`,
	}

	root.Flags().Bool("colon-filenames", false, "use this if you have filenames or directories with ':' in them - otherwise parsing will fail")
	root.Flags().Bool("blame", false, "use this to return direct links to blame view")
	return &root
}

// Are we executing as part of a pipeline?
func withinPipeline() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}
	return fi.Mode()&os.ModeNamedPipe != 0
}

func runCommand(cmd *cobra.Command, args []string) error {
	files := args
	if withinPipeline() {
		content, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		files = strings.Split(strings.TrimSpace(string(content)), "\n")
	}

	if len(files) == 0 {
		return cmd.Help()
	}

	var success bool
	colonFilenames, _ := cmd.Flags().GetBool("colon-filenames")
	blame, _ := cmd.Flags().GetBool("blame")

	// TODO: concurrency? Has a stair-step effect where the first git queries
	// of a new repo blocks the line
	for _, file := range files {
		filename, start, end := splitFilename(file, colonFilenames)

		url, err := getFileURL(filename, blame)
		if err != nil {
			fmt.Fprintln(os.Stderr, color.YellowString("%s: %v", file, err))
			continue
		}
		// We succeeded at least once
		success = true

		if start > 0 {
			url.Fragment = fmt.Sprintf("L%d", start)
		}

		if end > 0 {
			url.Fragment += fmt.Sprintf("-L%d", end)
		}
		fmt.Println(url)
	}

	// If no files were a success, exit with 1
	if !success {
		os.Exit(1)
	}

	return nil
}

var (
	filenameRe         = regexp.MustCompile(`(:[0-9\-]+){1,2}`)
	fallbackFilenameRe = regexp.MustCompile(`(:[0-9\-]+)+$`)
)

// splitFilename takes a filename that MAY have a start line number, end line
// number, and/or column number appended to the end.
//
// Returns those components split out.
//
// The goal is to match a start, and optional, end line number. Column number
// is parsed - but it will be ignored. This is to support streaming results
// from grep-like tools.
//
// Examples:
// path/to/file.txt:1
// path/to/file.txt:1-5
// path/to/file.txt:1:2
func splitFilename(text string, fallback bool) (string, int, int) {
	// The primary regex assumes no colons will be in the filepath.
	//
	// This is to support output from `rg` and other grep-like utilities that
	// have formats like "path/file:1: string that was matched".
	//
	// Colons are uncommon, so this is a decent trade-off. Allow it to be
	// circumvented.
	re := filenameRe
	if fallback {
		re = fallbackFilenameRe
	}

	loc := re.FindStringIndex(text)
	if loc == nil {
		return text, 0, 0
	}

	filename := text[:loc[0]]
	suffix := text[loc[0]:loc[1]]

	// Suffix is either...
	//  - :1
	//  - :1-5
	//  - :1:2
	suffix = strings.TrimLeft(suffix, ":")

	if strings.Count(suffix, ":") == 1 {
		s := strings.Split(suffix, ":")[0]
		start, err := strconv.Atoi(s)
		// TODO: This should probably not panic but I'm being lazy rn
		if err != nil {
			panic(err)
		}
		return filename, start, 0
	}

	if strings.Count(suffix, "-") == 1 {
		split := strings.Split(suffix, "-")

		start, err := strconv.Atoi(split[0])
		if err != nil {
			panic(err)
		}

		end, err := strconv.Atoi(split[1])
		if err != nil {
			panic(err)
		}

		return filename, start, end
	}

	start, err := strconv.Atoi(suffix)
	if err != nil {
		panic(err)
	}

	return filename, start, 0
}

func getFileURL(file string, blame bool) (*url.URL, error) {
	absFile, err := filepath.Abs(file)
	if err != nil {
		return nil, fmt.Errorf("couldn't locate absolute path to %s", file)
	}

	fileDir := filepath.Dir(absFile)
	workTree, err := Git.worktree(fileDir)
	if err != nil {
		return nil, err
	}

	rev, err := Git.upstreamRevision(workTree)
	if err != nil {
		return nil, err
	}

	baseURL, err := Git.baseURL(workTree)
	if err != nil {
		return nil, err
	}

	mode := "tree"
	if blame {
		mode = "blame"
	}
	revURL := baseURL.JoinPath(mode, rev)

	return revURL.JoinPath(strings.Replace(absFile, workTree, "", 1)), nil
}

var Git = git{
	map[string]string{},
	map[string]string{},
	map[string]*url.URL{},
}

type git struct {
	worktreeCache map[string]string
	revCache      map[string]string
	baseURLCache  map[string]*url.URL
}

// run a git command from the given working directory
func (g *git) run(cwd string, args ...string) (string, error) {
	args = append([]string{"-C", cwd}, args...)
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("'%s' failed: %w", cmd.String(), err)
	}
	return string(out), nil
}

func (g *git) worktree(cwd string) (string, error) {
	if dir, ok := g.worktreeCache[cwd]; ok {
		return dir, nil
	}

	var result string
	defer func() {
		if result != "" {
			g.worktreeCache[cwd] = result
		}
	}()

	gitDir, err := g.run(cwd, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	gitDir = strings.TrimSpace(gitDir)

	workTree, err := g.run(cwd, "config", "--get", "core.worktree")
	if err != nil {
		result = filepath.Dir(gitDir)
		return result, nil
	}
	workTree = strings.TrimSpace(workTree)

	// worktree is typically specified in git submodules
	if workTree == "" {
		result = filepath.Dir(gitDir)
		return result, nil
	}

	// when worktree is present, it's relative to the git conifg
	workTree, err = filepath.Abs(filepath.Join(gitDir, workTree))
	if err != nil {
		return "", err
	}

	result = workTree
	return result, nil
}

// upstreamRevision finds the most recent rev from HEAD that is upstream
//
// We do this by listing all upsream revisions, all revisions descending from
// HEAD, then finding the earliest commonality
func (g *git) upstreamRevision(cwd string) (string, error) {
	if rev, ok := g.revCache[cwd]; ok {
		return rev, nil
	}

	var result string
	defer func() {
		if result != "" {
			g.revCache[cwd] = result
		}
	}()

	localOnly, err := g.run(cwd, "log", "HEAD", "--oneline", "--not", "--remotes")
	if err != nil {
		return "", err
	}

	var selector string
	lines := strings.Split(strings.TrimSpace(localOnly), "\n")
	if len(lines) > 0 {
		line := lines[len(lines)-1]
		// <rev> <msg>
		selector = strings.Fields(line)[0] + "~1"
	} else {
		selector = "HEAD"
	}
	rev, err := g.run(cwd, "rev-list", "--abbrev-commit", "--abbrev=10", "--max-count=1", selector)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(rev), nil
}

func (g *git) baseURL(cwd string) (*url.URL, error) {
	if baseURL, ok := g.baseURLCache[cwd]; ok {
		return baseURL, nil
	}

	var result *url.URL
	defer func() {
		if result != nil {
			g.baseURLCache[cwd] = result
		}
	}()

	origin, err := g.run(cwd, "config", "--get", "remote.origin.url")
	if err != nil {
		return nil, err
	}

	origin = strings.TrimSpace(origin)
	origin = strings.TrimSuffix(origin, ".git")

	// TODO: support GHE
	if !strings.Contains(origin, "github.com") {
		return nil, fmt.Errorf("origin doesn't look like github.com - exiting: %s", origin)
	}

	// TODO: support GHE
	baseURL, err := url.Parse("https://github.com/")
	if err != nil {
		return nil, fmt.Errorf("internal error: %v", err)
	}

	if strings.HasPrefix(origin, "git@") {
		repo := strings.SplitN(origin, ":", 2)[1]
		repo = strings.TrimSuffix(repo, ".git")
		baseURL = baseURL.JoinPath(repo)
	} else if strings.HasPrefix(origin, "https") {
		repo := strings.SplitN(origin, "github.com/", 2)[1]
		baseURL = baseURL.JoinPath(repo)
	} else {
		return nil, fmt.Errorf("origin doesn't look like SSH or HTTPS: %s", origin)
	}

	result = baseURL
	return result, nil
}
