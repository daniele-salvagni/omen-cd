package omen

// HostTemplate is the starter host config printed by `omen init`.
const HostTemplate = `# omen host config
# Lives on the host, e.g. /etc/omen/main.yaml. Never committed.

repo: git@github.com:you/repo.git
dir: /srv/repo

# Optional. Defaults shown.
# branch: main
# source: .omen.yaml            # path inside the repo to the sync spec
# ssh_key: /etc/omen/id_ed25519
`

// SpecTemplate is the starter sync spec printed by `omen init --spec`.
const SpecTemplate = `# omen sync spec
# Lives in the repo at .omen.yaml (or the path set by 'source' in host config).

# Optional. Runs once after each successful non-empty deploy.
# Env: OMEN_SHA, OMEN_SHORT, OMEN_STATUS, OMEN_INSTANCE.
# notify: |
#   curl -fsS "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
#     -d chat_id="${TELEGRAM_CHAT_ID}" \
#     -d text="omen ${OMEN_STATUS} ${OMEN_SHORT}"

rules:
  - name: example
    paths: ["path/to/**"]
    run: echo changed
`
