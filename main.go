package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.Command("git", "log", "--format=%H\n%B-----DELIMITER-----")
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
	fmt.Println(hist.Markdown())
}

func (h History) Markdown() string {
	var sb strings.Builder
	if len(h.Features) > 1 {
		sb.WriteString("## Features\n\n")
		for _, feature := range h.Features {
			printCommit(&sb, feature)
		}
	}
	if len(h.Chores) > 1 {
		if len(h.Features) > 1 {
			sb.WriteString("\n")
		}
		sb.WriteString("## Chores\n\n")
		for _, feature := range h.Chores {
			printCommit(&sb, feature)
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
	default:
		h.Invalids = append(h.Invalids, c)
	}
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
	Invalids []Commit
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
	}

	return typ, parts[1]
}
