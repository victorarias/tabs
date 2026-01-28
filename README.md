# tabs 🎸

**Tablatures for AI Coding Sessions**

Share the technique behind your AI-assisted code, not just the output. Like guitar tabs teach you how to play a song, **tabs** captures and shares your prompts, debugging strategies, and architectural thinking with your team.

---

## What is tabs?

**tabs** is an open-source system for capturing, browsing, and sharing AI coding session transcripts from Claude Code and Cursor IDE.

**Why tabs?**
- 📝 **Prompts matter** - The conversation is often more valuable than the final code
- 🎓 **Learn by example** - See how experienced developers prompt, debug, and architect
- 🏠 **Local-first** - Capture everything locally, share only what you choose
- 🌐 **Transparency** - Shared sessions visible to everyone (team knowledge base)
- 🔒 **Privacy-conscious** - Review before sharing, no auto-uploads

---

## Features

- ✅ **Real-time capture** - Hook-based integration with Claude Code and Cursor
- ✅ **Local web UI** - Beautiful interface to browse and search your sessions
- ✅ **Remote server** - Self-hosted team knowledge base
- ✅ **Full context** - Captures prompts, responses, tool uses, thinking blocks
- ✅ **Search & filter** - Find relevant sessions by tags, dates, folders
- ✅ **API key management** - Secure authentication for uploads

---

## Quick Start

### Installation

```bash
# Download binaries (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/yourorg/tabs/main/install.sh | sh

# Or build from source
git clone https://github.com/yourorg/tabs
cd tabs
make install
```

### Setup

```bash
# Install hooks for Claude Code and Cursor
tabs-cli install

# Start local UI
tabs-cli ui
# Opens http://localhost:3787
```

### Configuration

```bash
# Configure remote server (optional)
tabs-cli config set server-url https://tabs.yourcompany.com
tabs-cli config set api-key tabs_your_api_key_here
```

---

## Architecture

**tabs** consists of four main components:

1. **tabs-cli** - Receives hook events and manages configuration
2. **tabs-daemon** - Captures sessions and writes to local JSONL files
3. **tabs-ui-local** - Local web UI for browsing sessions
4. **tabs-server** - Remote server for team sharing (optional)

```
┌─────────────────────────────────────────────┐
│  Claude Code / Cursor                       │
│         ↓ (hooks)                           │
│     tabs-cli                                │
│         ↓ (unix socket)                     │
│     tabs-daemon                             │
│         ↓                                   │
│  ~/.tabs/sessions/*.jsonl                   │
│         ↓                                   │
│  tabs-ui-local (localhost:3787)             │
│         ↓ (HTTPS)                           │
│  tabs-server (optional remote)              │
└─────────────────────────────────────────────┘
```

See [docs/02-system-architecture.md](docs/02-system-architecture.md) for details.

---

## Documentation

- **[Overview](docs/00-overview.md)** - Project overview and quick start
- **[Research Findings](docs/01-research-findings.md)** - How Claude Code & Cursor work
- **[System Architecture](docs/02-system-architecture.md)** - Component design and data flow
- **[Data Format](docs/03-data-format.md)** - JSONL and PostgreSQL schemas
- **[API Design](docs/04-api-design.md)** - Unix socket and HTTP APIs
- **[Local UI Flows](docs/05-local-ui-flows.md)** - Local web UI design
- **[Remote Server UX](docs/06-remote-server-ux.md)** - Remote server design
- **[Implementation Guide](docs/IMPLEMENTATION-READY.md)** - Step-by-step implementation

---

## Development

### Prerequisites

- Go 1.23+
- Node.js 20+
- PostgreSQL 16+ (for remote server)

### Building

```bash
# Build all components
make build

# Build specific component
make build-cli
make build-daemon
make build-server

# Build UI
cd ui
npm install
npm run build
```

### Running locally

```bash
# Terminal 1: Start daemon
./tabs-daemon

# Terminal 2: Install hooks
./tabs-cli install

# Terminal 3: Start local UI
cd ui
npm run dev
```

### Testing

```bash
# Go tests
make test

# E2E tests
make test-e2e
```

---

## Deployment

### Remote Server (Docker)

```bash
# Using Docker Compose
docker-compose up -d

# Or standalone
docker run -d \
  -p 8080:8080 \
  -e DATABASE_URL=postgresql://... \
  yourorg/tabs-server:latest
```

### Infrastructure

The remote server should be deployed behind an Identity-Aware Proxy (IAP):
- Cloudflare Access
- Google Cloud IAP
- Auth0
- OAuth2 Proxy

See [docs/02-system-architecture.md](docs/02-system-architecture.md#deployment) for details.

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to your fork (`git push origin feature/amazing-feature`)
7. Open a Pull Request

---

## Roadmap

### v0.1 - Local Capture (Claude Code)
- [x] Design complete
- [ ] Daemon with PID file management
- [ ] CLI with hook installation
- [ ] JSONL capture working
- [ ] Local storage established

### v0.2 - Local Web UI
- [ ] TanStack Start app
- [ ] Timeline view
- [ ] Session detail view
- [ ] Search functionality

### v0.3 - Remote Server
- [ ] Go HTTP server
- [ ] PostgreSQL schema
- [ ] API key management
- [ ] Session upload working
- [ ] Docker deployment

### v0.4 - Cursor Support
- [ ] Cursor hook handler
- [ ] SQLite poller
- [ ] End-to-end capture

### v1.0 - Production Ready
- [ ] Error handling
- [ ] Documentation
- [ ] CI/CD pipeline
- [ ] Release binaries
- [ ] Announcement

See [docs/00-overview.md](docs/00-overview.md#implementation-phases) for detailed phases.

---

## License

This project is licensed under the **GNU Affero General Public License v3.0 (AGPLv3)**.

This means:
- ✅ Free to use, modify, and distribute
- ✅ Can be used for commercial purposes
- ✅ Can be used internally without disclosure
- ⚠️ If you deploy as a network service (even internally), users must have access to source
- ⚠️ Any modifications must be shared under AGPLv3

See [LICENSE](LICENSE) for full text.

**Why AGPLv3?** We want tabs to remain open source and prevent proprietary forks, while still allowing commercial and internal use.

---

## Inspiration

- [SpecStory](https://specstory.com/) - Original inspiration for prompt sharing
- [getspecstory](https://github.com/specstoryai/getspecstory) - Provider pattern architecture
- Guitar tabs - Sharing technique, not just the song

---

## Support

- **Documentation:** [docs/](docs/)
- **Issues:** [GitHub Issues](https://github.com/yourorg/tabs/issues)
- **Discussions:** [GitHub Discussions](https://github.com/yourorg/tabs/discussions)

---

## Credits

Built with ❤️ by developers who believe in transparent knowledge sharing.

**Powered by:**
- [Claude Code](https://claude.ai/code) - AI pair programmer
- [Cursor](https://cursor.com/) - AI-first code editor
- [Go](https://go.dev/) - Backend language
- [TanStack Start](https://tanstack.com/start) - Frontend framework
- [PostgreSQL](https://postgresql.org/) - Database

---

**Share your tabs, share your craft.** 🎸
