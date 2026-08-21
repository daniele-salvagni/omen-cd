package omen

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

// SpecTemplate is the starter sync spec printed by `omen init spec`.
//
//go:embed templates/spec.yaml
var SpecTemplate string

//go:embed templates/service.tmpl
var serviceTmpl string

//go:embed templates/timer.tmpl
var timerTmpl string

// HostInit holds the values used to render a host config starter. Every
// field is optional; when unset, the corresponding line is emitted commented
// with its default value.
type HostInit struct {
	Repo, Dir, Branch, Source, User, SSHKey, Interval string
}

// RenderHostInit prints the host config template with any provided values
// substituted inline. Unset fields stay commented with a `# default` hint.
func RenderHostInit(h HostInit) string {
	var b strings.Builder
	b.WriteString("# omen host config\n")
	b.WriteString("# Lives on the host, e.g. /etc/omen/<name>.yaml. Never committed.\n\n")

	repo := h.Repo
	if repo == "" {
		repo = "git@github.com:you/repo.git"
	}
	dir := h.Dir
	if dir == "" {
		dir = "/srv/repo"
	}
	fmt.Fprintf(&b, "repo: %s\n", repo)
	fmt.Fprintf(&b, "dir: %s\n\n", dir)

	optional := []struct{ field, def, comment, val string }{
		{"branch", "main", "default", h.Branch},
		{"source", ".omen.yaml", "spec path inside the repo", h.Source},
		{"user", "alice", "run service as this user (default: root)", h.User},
		{"ssh_key", "/etc/omen/id_ed25519", "", h.SSHKey},
		{"interval", "60s", "timer cadence", h.Interval},
	}
	for _, o := range optional {
		if o.val != "" {
			fmt.Fprintf(&b, "%s: %s\n", o.field, o.val)
			continue
		}
		if o.comment != "" {
			fmt.Fprintf(&b, "# %s: %-25s # %s\n", o.field, o.def, o.comment)
		} else {
			fmt.Fprintf(&b, "# %s: %s\n", o.field, o.def)
		}
	}
	return b.String()
}

// UnitData is the data passed to service/timer templates.
type UnitData struct {
	Name       string
	ConfigPath string
	User       string
	Interval   string
}

// RenderServiceUnit produces a systemd service unit for the given host,
// derived from its config path and fields.
func RenderServiceUnit(h *Host, name, configPath string) (string, error) {
	return renderTemplate(serviceTmpl, UnitData{
		Name:       name,
		ConfigPath: configPath,
		User:       h.User,
		Interval:   h.Interval,
	})
}

// RenderTimerUnit produces a systemd timer unit for the given host.
func RenderTimerUnit(h *Host, name string) (string, error) {
	return renderTemplate(timerTmpl, UnitData{
		Name:     name,
		User:     h.User,
		Interval: h.Interval,
	})
}

func renderTemplate(tmpl string, data UnitData) (string, error) {
	t, err := template.New("unit").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}
