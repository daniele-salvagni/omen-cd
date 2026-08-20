package omen

import _ "embed"

// HostTemplate is the starter host config printed by `omen init`.
//
//go:embed templates/host.yaml
var HostTemplate string

// SpecTemplate is the starter sync spec printed by `omen init --spec`.
//
//go:embed templates/spec.yaml
var SpecTemplate string
