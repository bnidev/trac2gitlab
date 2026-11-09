// Package references rewrites Trac ticket and changeset references for GitLab.
package references

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Options struct {
	Enabled        bool
	TicketsFile    string
	ChangesetsFile string
	BaseURL        string
	ProjectPath    string
}

type Refs struct {
	enabled     bool
	tickets     map[int64]int64
	changesets  map[string]string
	baseURL     string
	projectPath string
}

type ticketFile struct {
	Tickets map[string]int64 `json:"tickets"`
}

type changesetFile struct {
	Changesets map[string]string `json:"changesets"`
}

func Load(opts Options) (*Refs, error) {
	r := &Refs{
		enabled:     opts.Enabled,
		tickets:     make(map[int64]int64),
		changesets:  make(map[string]string),
		baseURL:     strings.TrimRight(opts.BaseURL, "/"),
		projectPath: strings.Trim(opts.ProjectPath, "/"),
	}
	if !opts.Enabled {
		return r, nil
	}
	if err := loadTickets(opts.TicketsFile, r.tickets); err != nil {
		return nil, fmt.Errorf("load ticket mapping: %w", err)
	}
	if err := loadChangesets(opts.ChangesetsFile, r.changesets); err != nil {
		return nil, fmt.Errorf("load changeset mapping: %w", err)
	}
	return r, nil
}

func loadTickets(path string, out map[int64]int64) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var file ticketFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}
	for key, iid := range file.Tickets {
		id, err := strconv.ParseInt(key, 10, 64)
		if err == nil {
			out[id] = iid
		}
	}
	return nil
}

func loadChangesets(path string, out map[string]string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var file changesetFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}
	for hg, git := range file.Changesets {
		out[strings.ToLower(hg)] = git
	}
	return nil
}

func (r *Refs) Rewrite(text string) string {
	if r == nil || !r.enabled || text == "" {
		return text
	}
	// Leave fenced code blocks untouched.
	parts := strings.Split(text, "```")
	for i := range parts {
		if i%2 == 0 {
			parts[i] = r.rewriteText(parts[i])
		}
	}
	return strings.Join(parts, "```")
}

var (
	ticketBracketRe  = regexp.MustCompile(`(?i)\[ticket:#?(\d+)\]`)
	ticketPrefixRe   = regexp.MustCompile(`(?i)\bticket:#?(\d+)\b`)
	ticketRawRe      = regexp.MustCompile(`(\A|[^A-Za-z0-9_/&#])#(\d+)\b`)
	ticketMarkerRe   = regexp.MustCompile("\x00T(\\d+)\x00")
	changesetBracket = regexp.MustCompile(`(?i)\[changeset:([0-9a-f]{6,40})\]`)
	changesetRe      = regexp.MustCompile(`(?i)\bchangeset:([0-9a-f]{6,40})\b`)
	bracketHashRe    = regexp.MustCompile(`\[([0-9a-f]{6,40})\]`)
	changesetMarker  = regexp.MustCompile("\x00C(\\d+)\x00")
)

func (r *Refs) rewriteText(text string) string {
	// Replace explicit ticket forms first, using a marker so the raw # pass
	// cannot rewrite an already-rewritten ID a second time.
	text = ticketBracketRe.ReplaceAllStringFunc(text, func(m string) string {
		id := ticketBracketRe.FindStringSubmatch(m)[1]
		if iid, ok := r.ticketIID(id); ok {
			return marker(iid)
		}
		return m
	})
	text = ticketPrefixRe.ReplaceAllStringFunc(text, func(m string) string {
		id := ticketPrefixRe.FindStringSubmatch(m)[1]
		if iid, ok := r.ticketIID(id); ok {
			return marker(iid)
		}
		return m
	})
	text = ticketRawRe.ReplaceAllStringFunc(text, func(m string) string {
		sub := ticketRawRe.FindStringSubmatch(m)
		if iid, ok := r.ticketIID(sub[2]); ok {
			return sub[1] + marker(iid)
		}
		return m
	})
	text = ticketMarkerRe.ReplaceAllString(text, "#$1")

	// Protect generated links so later patterns cannot rewrite them again.
	var links []string
	protect := func(link string) string {
		links = append(links, link)
		return fmt.Sprintf("\x00C%d\x00", len(links)-1)
	}
	text = changesetBracket.ReplaceAllStringFunc(text, func(m string) string {
		ref := changesetBracket.FindStringSubmatch(m)[1]
		if link, ok := r.changesetLink(ref); ok {
			return protect(link)
		}
		return m
	})
	text = changesetRe.ReplaceAllStringFunc(text, func(m string) string {
		ref := changesetRe.FindStringSubmatch(m)[1]
		if link, ok := r.changesetLink(ref); ok {
			return protect(link)
		}
		return m
	})
	text = bracketHashRe.ReplaceAllStringFunc(text, func(m string) string {
		ref := bracketHashRe.FindStringSubmatch(m)[1]
		if link, ok := r.changesetLink(ref); ok {
			return protect(link)
		}
		return m
	})
	text = changesetMarker.ReplaceAllStringFunc(text, func(m string) string {
		idx, _ := strconv.Atoi(changesetMarker.FindStringSubmatch(m)[1])
		return links[idx]
	})
	return text
}

func marker(iid int64) string {
	return "\x00T" + strconv.FormatInt(iid, 10) + "\x00"
}

func (r *Refs) ticketIID(id string) (int64, bool) {
	tid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, false
	}
	iid, ok := r.tickets[tid]
	return iid, ok
}

func (r *Refs) changesetLink(ref string) (string, bool) {
	sha, ok := r.resolveChangeset(ref)
	if !ok {
		return "", false
	}
	label := ref
	if len(label) > 8 {
		label = label[:8]
	}
	if r.baseURL == "" || r.projectPath == "" {
		return label, true
	}
	return fmt.Sprintf("[%s](%s/%s/-/commit/%s)", label, r.baseURL, r.projectPath, sha), true
}

func (r *Refs) resolveChangeset(ref string) (string, bool) {
	ref = strings.ToLower(ref)
	if sha, ok := r.changesets[ref]; ok {
		return sha, true
	}
	if len(ref) < 6 {
		return "", false
	}
	for hg, sha := range r.changesets {
		if strings.HasPrefix(hg, ref) {
			return sha, true
		}
	}
	return "", false
}
