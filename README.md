<p align="center">
  <img src="docs/omen.png" alt="omen" height="96px">
</p>

# omen

A pull-based GitOps runner. Watches a branch, runs shell commands when files
change.

```
pull --> match --> run --> notify
```

## Install

Prebuilt binaries for `linux/amd64`, `linux/arm64`, and `linux/arm/v7`:

```sh
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/;s/armv7l/armv7/')
curl -fsSL https://github.com/daniele-salvagni/omen-cd/releases/latest/download/omen-linux-${ARCH} | sudo install -m 755 /dev/stdin /usr/local/bin/omen
```

Or from source (requires Go):

```sh
go install github.com/daniele-salvagni/omen-cd/cmd/omen@latest
```

## Quickstart

```sh
# 1. Host config
sudo mkdir -p /etc/omen
omen init host | sudo tee /etc/omen/main.yaml
sudo $EDITOR /etc/omen/main.yaml   # set repo and dir

# 2. Sync spec, in your repo root
omen init spec > .omen.yaml
$EDITOR .omen.yaml                 # define rules
git add .omen.yaml && git commit -m 'omen spec' && git push

# 3. Systemd
omen unit service | sudo tee /etc/systemd/system/omen@.service
omen unit timer   | sudo tee /etc/systemd/system/omen@.timer
sudo systemctl enable --now omen@main.timer

# 4. First deploy
sudo omen --config /etc/omen/main.yaml --apply-all
```

`git push` handles every deploy after that.

omen is a one-shot binary; any scheduler works. systemd is the default because
it captures logs and status cleanly.

## Configure

Two files. `omen init host` and `omen init spec` print starters.

**Host config** at `/etc/omen/<instance>.yaml`:

```yaml
repo: git@github.com:you/infra.git
dir: /srv/infra
# branch: main                    # default
# source: .omen.yaml              # spec path inside the repo
# ssh_key: /etc/omen/id_ed25519
```

`omen@<name>.timer` reads `/etc/omen/<name>.yaml`. Enable multiple timers
(`omen@web.timer`, `omen@db.timer`) to run multiple instances on one host.

**Sync spec** in the repo (default `.omen.yaml` at the root):

```yaml
notify: |
  curl -fsS "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
    -d chat_id="${TELEGRAM_CHAT_ID}" \
    -d text="omen ${OMEN_STATUS} ${OMEN_SHORT}"

rules:
  - name: pangolin
    paths: ["stacks/pangolin/**"]
    run: cd stacks/pangolin && docker compose up -d

  - name: cron jobs
    paths: ["cron.d/**"]
    run: install -m 644 cron.d/* /etc/cron.d/
```

Every rule whose globs match a changed file runs, in order. `notify` runs once
per successful deploy with `OMEN_SHA`, `OMEN_SHORT`, `OMEN_STATUS`, and
`OMEN_INSTANCE` set.

Secrets go in `/etc/omen/<instance>.env`, loaded by systemd, `chmod 600`.

Globs use [doublestar](https://github.com/bmatcuk/doublestar) syntax, so `**`
spans directory boundaries.

## Operating

- `systemctl status omen@main.timer`: next run, last activation.
- `systemctl start omen@main.service`: trigger a sync now.
- `journalctl -u omen@main.service -f`: tail deploy logs.

## Commands

```
omen [--config PATH] [--dry-run] [--apply-all]
omen init [host|spec]
omen unit [service|timer]
omen version
```

- `--dry-run` prints the pending diff and matched rules, writes and runs
  nothing.
- `--apply-all` treats every tracked file as changed for one invocation.

## Behavior

- First run with no state records HEAD and executes no rules, so nothing runs
  unexpectedly on install. `--apply-all` forces a full initial deploy.
- Fast-forward only. Divergence from the remote aborts the sync loudly.
- State (the last applied sha) advances only after every matching rule succeeds.
  A mid-batch failure leaves state at the previous sha; the next tick retries
  the same diff. Rules must be idempotent.
- `notify` fires once with `OMEN_STATUS=deployed: <names>` on success or
  `failed: <rule>` on error.
- A failing `notify` is logged and ignored.
- One flock per instance prevents timer and manual runs from colliding.

## License

Apache 2.0.
