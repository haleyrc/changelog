package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	currentContents, err := ioutil.ReadFile("CHANGELOG.md")
	if err != nil {
		panic(err)
	}

	lastTag := getLastTag()
	rng := fmt.Sprintf("%s..HEAD", lastTag)
	cmd := exec.Command("git", "log", rng, "--format=%H\n%B-----DELIMITER-----")
	contents, err := cmd.Output()
	if err != nil {
		panic(err)
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

	newTag := calculateNewTag(lastTag, hist)
	fmt.Println("New tag:", newTag)
	newContents := fmt.Sprintf(
		"# Version %s (%s)\n\n",
		newTag,
		time.Now().Format(time.RFC3339),
	)
	newContents += hist.Markdown() + "\n" + string(currentContents)
	fmt.Println(ioutil.WriteFile("CHANGELOG.md", []byte(newContents), os.ModePerm))
}

func (h History) Markdown() string {
	var sb strings.Builder
	if len(h.Breaks) > 0 {
		sb.WriteString("## Breaking Changes\n\n")
		for _, breaker := range h.Breaks {
			printCommit(&sb, breaker)
		}
	}
	if len(h.Fixes) > 0 {
		if len(h.Breaks) > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("## Bug Fixes\n\n")
		for _, fix := range h.Fixes {
			printCommit(&sb, fix)
		}
	}
	if len(h.Features) > 0 {
		if len(h.Fixes) > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("## Features\n\n")
		for _, feature := range h.Features {
			printCommit(&sb, feature)
		}
	}
	if len(h.Chores) > 0 {
		if len(h.Features) > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("## Chores\n\n")
		for _, chore := range h.Chores {
			printCommit(&sb, chore)
		}
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
	default:
		h.Invalids = append(h.Invalids, c)
	}
}

func splitTag(t string) (int, int, int) {
	parts := strings.Split(t[1:], ".")
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

func printCommit(sb *strings.Builder, c Commit) {
	s := fmt.Sprintf(
		"* %s ([%s](%s%s)\n",
		c.Subject,
		c.Hash[:6],
		"https://github.com/haleyrc/changelog/commit/",
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
	}

	return typ, parts[1]
}

func getLastTag() string {
	cmd := exec.Command("git", "describe", "--abbrev=0", "--tags", "--always")
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}
