package omen

import _ "embed"

// HostTemplate is the starter host config printed by `omen init host`.
//
//go:embed templates/host.yaml
var HostTemplate string

// SpecTemplate is the starter sync spec printed by `omen init spec`.
//
//go:embed templates/spec.yaml
var SpecTemplate string

// ServiceUnit is the systemd service unit printed by `omen unit service`.
//
//go:embed templates/omen@.service
var ServiceUnit string

// TimerUnit is the systemd timer unit printed by `omen unit timer`.
//
//go:embed templates/omen@.timer
var TimerUnit string
