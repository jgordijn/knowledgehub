# KnowledgeHub

A personal knowledge radar that monitors RSS feeds and blogs, summarizes articles with AI, learns your preferences through star ratings, and serves a mobile-friendly feed — all in a single binary.

## Features

- **RSS & blog monitoring** — add RSS feeds or blog URLs, KnowledgeHub checks them every 30 minutes
- **AI summaries & scoring** — each article gets a 2-4 sentence summary and 1-5 star relevance rating via OpenRouter (Claude, GPT, Llama, etc.)
- **Preference learning** — rate articles yourself and the AI learns what you care about over time
- **Article chat** — ask questions about any article in a streaming chat panel
- **Quarantine** — broken feeds are automatically quarantined after 5 consecutive failures
- **Mobile-friendly** — responsive Tailwind CSS design, works great on phone browsers
- **Single binary** — Go backend with SvelteKit frontend embedded, just copy and run
- **Browser notifications** — get notified when 5-star articles arrive (via PocketBase realtime SSE)

## Prerequisites

For **building** from source:
- Go 1.24+
- Bun (for building the frontend)

For **running** the binary:
- Linux amd64 (the release target)
- An OpenRouter API key ([get one here](https://openrouter.ai/keys))

## Quick Start

### Build from source

```bash
git clone https://github.com/jgordijn/knowledgehub.git
cd knowledgehub
make build
```

This builds the SvelteKit frontend, embeds it in the Go binary, and outputs `build/knowledgehub`.

### Run it

```bash
./build/knowledgehub serve
```

Open http://localhost:8090 in your browser.

On first launch, PocketBase will show you a link to create your superuser account. You can also create one from the command line:

```bash
./build/knowledgehub superuser upsert you@example.com yourpassword
```

### Configure

1. Log in to the app
2. Go to **Settings** (bottom nav)
3. Enter your **OpenRouter API key** and choose a **model** (default: `anthropic/claude-sonnet-4`)
4. Go to **Resources** and add your first RSS feed or blog URL

That's it. KnowledgeHub will start fetching and summarizing articles automatically.

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KH_DATA_DIR` | `./kh_data` | Directory for the SQLite database and PocketBase data |

### Command Line Flags

```bash
./knowledgehub serve --http=0.0.0.0:8090   # Listen address and port
```

All PocketBase CLI flags are available — run `./knowledgehub --help` for the full list.

### Settings (in the UI)

| Setting | Description |
|---------|-------------|
| OpenRouter API Key | Your `sk-or-v1-...` key from openrouter.ai |
| Model | OpenRouter model ID, e.g. `anthropic/claude-sonnet-4`, `openai/gpt-4o`, `meta-llama/llama-3.1-70b-instruct` |

## Install as a System Service

### Set up the host

```bash
# Create a system user
sudo useradd -r -s /bin/false knowledgehub

# Create the install directory
sudo mkdir -p /opt/knowledgehub/data
sudo chown -R knowledgehub:knowledgehub /opt/knowledgehub
```

### Deploy

```bash
# Download the latest release (or build with: make release)
curl -LO https://github.com/jgordijn/knowledgehub/releases/latest/download/knowledgehub-linux-amd64.tar.gz

# Copy to the target machine (skip if downloading directly on the host)
scp knowledgehub-linux-amd64.tar.gz yourhost:/tmp/

# On the target machine
ssh yourhost
cd /opt/knowledgehub
sudo tar xzf /tmp/knowledgehub-linux-amd64.tar.gz
sudo cp knowledgehub.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now knowledgehub
```

### Verify

```bash
sudo systemctl status knowledgehub
curl http://localhost:8090    # should return the UI
```

## Updating

### Automatic updates

The release tarball includes an auto-updater that checks GitHub for new releases every minute and installs them automatically.

```bash
cd /opt/knowledgehub
sudo cp knowledgehub-updater.sh /opt/knowledgehub/
sudo cp knowledgehub-updater.service knowledgehub-updater.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now knowledgehub-updater.timer
```

Check update status:

```bash
systemctl status knowledgehub-updater.timer    # next/last run
journalctl -u knowledgehub-updater.service     # update logs
```

### Manual updates

```bash
sudo systemctl stop knowledgehub
cd /opt/knowledgehub
curl -LO https://github.com/jgordijn/knowledgehub/releases/latest/download/knowledgehub
chmod +x knowledgehub
sudo systemctl start knowledgehub
```

The SQLite database in `/opt/knowledgehub/data` is preserved across updates. PocketBase handles schema migrations automatically.

## Development

```bash
# Run tests
make test

# Start the backend in dev mode (serves on :8090)
make dev

# Frontend dev with hot reload (separate terminal)
cd ui && bun run dev
```

### Project layout

```
cmd/knowledgehub/      Go entry point, PocketBase collections and hooks
internal/ai/           OpenRouter client, summarizer, preference learning
internal/engine/       Scheduler, RSS parser, scraper, readability, quarantine
internal/routes/       Custom API endpoints (article chat)
internal/testutil/     Shared test helpers
ui/                    SvelteKit frontend (Tailwind CSS, static adapter)
```

## PocketBase Admin

The PocketBase admin dashboard is always available at `http://localhost:8090/_/`. From there you can:

- Browse and edit all data directly
- Manage auth (your superuser account)
- View logs
- Export/import collections

## License

Personal project — not published under a public license.
