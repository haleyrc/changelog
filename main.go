package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/haleyrc/changelog/git"
)

func main() {
	ctx := context.Background()

	if _, err := os.Stat(".git"); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Not in a git repository")
		} else {
			fmt.Printf("Failed to determine find git data: %v\n", err)
		}
		os.Exit(1)
	}

	repo, err := getRepo()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	currentContents, err := readChangelog()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	lastTag := getLastTag()
	rng := fmt.Sprintf("%s..HEAD", lastTag)
	if lastTag == "" {
		rng = "HEAD"
	}

	contents, err := getLog(rng)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	commitLines := strings.Split(string(contents), "-----DELIMITER-----")
	var hist History
	for _, lines := range commitLines {
		commit, ok := splitCommit(lines)
		if !ok {
			continue
		}
		hist.Add(commit)
	}

	if hist.Empty() {
		fmt.Println("No changes to be recorded")
		os.Exit(0)
	}

	newTag := calculateNewTag(lastTag, hist)
	newContents := fmt.Sprintf(
		"## [Version %s](https://github.com/%s/releases/tag/%s) (%s)\n\n",
		newTag,
		repo,
		newTag,
		time.Now().Format("2006-01-02 15:04"),
	)
	newContents += hist.Markdown(repo) + "\n" + string(currentContents)

	if err := ioutil.WriteFile("CHANGELOG.md", []byte(newContents), os.ModePerm); err != nil {
		fmt.Printf("Failed to write CHANGELOG.md: %v\n", err)
		os.Exit(1)
	}

	if err := git.Add(ctx, "CHANGELOG.md"); err != nil {
		fmt.Printf("Failed to add CHANGELOG.md to commit: %v\n", err)
		os.Exit(1)
	}

	commitMsg := fmt.Sprintf("Update CHANGELOG for %s", newTag)
	if err := git.Commit(ctx, commitMsg); err != nil {
		fmt.Printf("Failed to commit CHANGELOG.md: %v\n", err)
		os.Exit(1)
	}

	tagMsg := fmt.Sprintf("Tag for %s", newTag)
	if err := git.Tag(ctx, newTag, tagMsg); err != nil {
		fmt.Printf("Failed to tag commit: %v\n", err)
		os.Exit(1)
	}
}

func readChangelog() (string, error) {
	_, err := os.Stat("CHANGELOG.md")
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		return "", nil
	}
	b, err := ioutil.ReadFile("CHANGELOG.md")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (h History) Markdown(repo string) string {
	var sb strings.Builder
	if len(h.Breaks) > 0 {
		sb.WriteString("### Breaking Changes\n\n")
		for _, breaker := range h.Breaks {
			printCommit(&sb, repo, breaker)
		}
		sb.WriteString("\n")
	}
	if len(h.Fixes) > 0 {
		sb.WriteString("### Bug Fixes\n\n")
		for _, fix := range h.Fixes {
			printCommit(&sb, repo, fix)
		}
		sb.WriteString("\n")
	}
	if len(h.Features) > 0 {
		sb.WriteString("### Features\n\n")
		for _, feature := range h.Features {
			printCommit(&sb, repo, feature)
		}
		sb.WriteString("\n")
	}
	if len(h.Chores) > 0 {
		sb.WriteString("### Chores\n\n")
		for _, chore := range h.Chores {
			printCommit(&sb, repo, chore)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (h *History) Add(c Commit) {
	switch c.Type {
	case Feature:
		h.Features = append(h.Features, c)
	case Chore:
		h.Chores = append(h.Chores, c)
	case Fix:
		h.Fixes = append(h.Fixes, c)
	case Break:
		h.Breaks = append(h.Breaks, c)
	case Docs:
		h.Docs = append(h.Docs, c)
	case Build:
		h.Build = append(h.Build, c)
	default:
		h.Invalids = append(h.Invalids, c)
	}
}

func splitTag(t string) (int, int, int) {
	if t == "" {
		return 0, 0, 0
	}
	parts := strings.Split(t[1:], ".")
	if len(parts) < 3 {
		return 0, 0, 0
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	return major, minor, patch
}

func calculateNewTag(ct string, h History) string {
	major, minor, patch := splitTag(ct)
	switch {
	case len(h.Breaks) > 0:
		major, minor, patch = major+1, 0, 0
	case len(h.Features) > 0:
		minor, patch = minor+1, 0
	default:
		patch = patch + 1
	}
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}

func printCommit(sb *strings.Builder, repo string, c Commit) {
	s := fmt.Sprintf(
		"* %s ([%s](https://github.com/%s/%s))\n",
		c.Subject,
		c.Hash[:6],
		repo,
		c.Hash,
	)
	sb.WriteString(s)
}

type History struct {
	Features []Commit
	Chores   []Commit
	Fixes    []Commit
	Invalids []Commit
	Breaks   []Commit
	Build    []Commit
	Docs     []Commit
}

func (h History) Empty() bool {
	return (h.Features == nil || len(h.Features) == 0) &&
		(h.Chores == nil || len(h.Chores) == 0) &&
		(h.Fixes == nil || len(h.Fixes) == 0) &&
		(h.Breaks == nil || len(h.Breaks) == 0)
}

type Commit struct {
	Hash    string
	Type    CommitType
	Subject string
}

type CommitType int

const (
	Invalid CommitType = iota
	Feature
	Chore
	Fix
	Break
	Docs
	Build
)

func splitCommit(s string) (Commit, bool) {
	var c Commit
	lines := strings.Split(
		strings.TrimSpace(s),
		"\n",
	)

	c.Hash = lines[0]

	typ, sub := parseMessage(lines[1:])
	if typ == Invalid {
		return Commit{}, false
	}

	c.Type = typ
	c.Subject = sub

	return c, true
}

func parseMessage(ls []string) (CommitType, string) {
	if len(ls) < 1 {
		return Invalid, ""
	}
	// The rest of the lines should be the
	// commit message detail if there is any
	return parseSubject(ls[0])
}

func parseSubject(s string) (CommitType, string) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		return Invalid, ""
	}

	prefix := strings.ToLower(strings.TrimSpace(parts[0]))
	typ := Invalid
	switch {
	case strings.HasPrefix(prefix, "feature") ||
		strings.HasPrefix(prefix, "feat"):
		typ = Feature
	case strings.HasPrefix(prefix, "chore"):
		typ = Chore
	case strings.HasPrefix(prefix, "fix"):
		typ = Fix
	case strings.HasPrefix(prefix, "break"):
		typ = Break
	case strings.HasPrefix(prefix, "docs"):
		typ = Docs
	case strings.HasPrefix(prefix, "build"):
		typ = Build
	}

	return typ, parts[1]
}

func getLastTag() string {
	cmd := exec.Command("git", "describe", "--abbrev=0", "--tags")
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func getLog(rng string) ([]byte, error) {

	cmd := exec.Command("git", "log", rng, "--format=%H\n%B-----DELIMITER-----")
	var errBuff bytes.Buffer
	cmd.Stderr = &errBuff
	contents, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Failed to get repo name: %v\n%s", err, errBuff.String())
	}
	return contents, nil
}

func getRepo() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	var errBuff bytes.Buffer
	cmd.Stderr = &errBuff
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Failed to get repo name: %v\n%s", err, errBuff.String())
	}
	repo := string(out)
	repo = strings.TrimSpace(repo)
	repo = strings.TrimPrefix(repo, "git@github.com:")
	repo = strings.TrimSuffix(repo, ".git")
	return repo, nil
}
