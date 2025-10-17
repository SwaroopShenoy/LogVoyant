# 🔍 LogVoyant

**AI-powered log analysis in your browser. Zero setup, works offline.**

Stop grepping through logs. LogVoyant auto-discovers your logs, analyzes them for errors, and suggests fixes—all in a single binary with zero configuration.

```bash
curl -sSL https://logvoyant.sh | sh
logvoyant start
# → Open http://localhost:3100
```

---

## Why LogVoyant?

**Not another enterprise monitoring platform.** LogVoyant is built for developers who need quick answers, not complex dashboards.

- **Zero Setup** - No YAML. No agents. No database. Just run it.
- **Works Offline** - Pattern matching built-in. LLM analysis optional.
- **Context-Aware** - Remembers past issues. Connects the dots across analyses.
- **Local-First** - Your logs stay on your machine. No cloud required.
- **One Binary** - 10MB, no dependencies, runs anywhere.

Perfect for:
- 🐛 Debugging failed deployments
- 🔥 Troubleshooting prod incidents locally
- 📚 Learning from past issues
- ⚡ Quick log analysis without Datadog bills

---

## Features

### 🎯 Auto-Discovery
Automatically finds and tails logs from:
- Kubernetes pods (via `kubectl`)
- Docker containers
- Local files (`/var/log/*`)

### 🧠 Smart Analysis
- **Pattern Matching**: 20+ common errors (OOMKilled, CrashLoopBackOff, timeouts)
- **LLM Analysis**: Optional AI-powered root cause analysis (free Groq API)
- **Historical Context**: Tracks recurring issues and suggests fixes based on past analyses

### 📊 Clean Web UI
- Live log streaming
- Side-by-side logs and analysis
- Timeline of past issues
- Mark issues as resolved

### 🚀 Zero Config
```bash
logvoyant start
# That's it. UI opens at localhost:3100
```

---

## Quick Start

### Install
```bash
# macOS / Linux
curl -sSL https://logvoyant.sh | sh

# Or download from releases
# https://github.com/SwaroopShenoy/logvoyant/releases
```

### Run
```bash
logvoyant start

# With AI analysis (optional)
export GROQ_API_KEY=your_key_here  # Get free key at groq.com
logvoyant start

# Custom port
logvoyant start --port 8080
```

### Use
1. Open `http://localhost:3100`
2. Select a log stream
3. Click **Analyze** to get insights
4. See root cause + suggested fixes

---

## How It Works

```
Your Logs → Auto-Discover → Tail → Analyze → Show Insights
                ↓            ↓        ↓
            kubectl      Stream   Pattern
            docker        to UI    Matching
            files                    +
                                  LLM (opt)
```

**Context Tracking:**
Every analysis is stored as context. Future analyses reference past issues to give you better insights over time.

Example:
```
Analysis 1 (10:00 AM): "DB_HOST env var missing"
Analysis 2 (11:00 AM): "Still timing out - network policy issue"
                       ↳ References previous DB_HOST context
```

---

## Screenshots

*Coming soon - UI is being built!*

---

## Comparison

| Feature | LogVoyant | Datadog | ELK Stack | grep |
|---------|-----------|---------|-----------|------|
| Setup time | 10 seconds | Hours | Days | 0s |
| Cost | Free | $$$$$ | Self-host | Free |
| AI Analysis | ✅ | ✅ | ❌ | ❌ |
| Works Offline | ✅ | ❌ | ✅ | ✅ |
| Context Tracking | ✅ | ✅ | ❌ | ❌ |
| Local-first | ✅ | ❌ | ✅ | ✅ |

---

## Roadmap

### v0.1 (MVP) - In-progress
- [x] Web UI
- [x] Auto-discovery (kubectl, docker, files)
- [x] Pattern matching
- [ ] LLM integration (Groq)
- [ ] Context tracking

### v0.2
- [ ] TUI mode (terminal UI)
- [ ] Multiple LLM providers (Claude, GPT)
- [ ] Export reports (Markdown, JSON)
- [ ] Anomaly detection (error rate spikes)

### v0.3
- [ ] Valkey/Redis backend (persistent storage)
- [ ] Notifications
- [ ] Custom pattern library
- [ ] Multi-cluster support

---

## Configuration

LogVoyant works with zero config, but you can customize it:

```yaml
# ~/.logvoyant/config.yaml (optional)

sources:
  - type: kubectl
    namespaces: ["prod", "staging"]
  - type: docker
  - type: file
    paths: ["/var/log/app/*.log"]

analyzer:
  provider: groq  # or claude, openai
  api_key: ${GROQ_API_KEY}
  fallback_enabled: true

storage:
  path: ~/.logvoyant/logs.db
  max_lines_per_stream: 10000
```

---

## Development

```bash
# Clone
git clone https://github.com/SwaroopShenoy/logvoyant
cd logvoyant

# Run
go run main.go

# Build
go build -o logvoyant main.go

```

---

## Contributing

PRs welcome! This is a learning project and community contributions are encouraged.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

Apache License 2.0 - see [LICENSE](LICENSE)

**Copyright © 2025 Swaroop Shenoy**

---

## Credits

Built by [@SwaroopShenoy](https://github.com/SwaroopShenoy)

Inspired by frustration with complex log analysis tools.

---

## Support

- 🐛 [Report a bug](https://github.com/SwaroopShenoy/logvoyant/issues)
- 💡 [Request a feature](https://github.com/SwaroopShenoy/logvoyant/issues)
- 💬 [Discussions](https://github.com/SwaroopShenoy/logvoyant/discussions)

---

**Star ⭐ this repo if LogVoyant helped you debug something!**