# Contributing to LogVoyant

Thanks for considering contributing! LogVoyant is a learning project and all contributions are welcome.

## How to Contribute

### Reporting Bugs
1. Check if the bug is already reported in [Issues](https://github.com/SwaroopShenoy/logvoyant/issues)
2. If not, open a new issue with:
   - Clear title
   - Steps to reproduce
   - Expected vs actual behavior
   - Your environment (OS, Go version)
   - Logs if applicable

### Suggesting Features
1. Open an issue with the `enhancement` label
2. Describe the use case
3. Explain why it would be useful
4. (Optional) Suggest implementation approach

### Code Contributions

#### First Time?
Look for issues labeled `good first issue` - these are beginner-friendly tasks.

#### Process
1. Fork the repo
2. Create a branch: `git checkout -b feature/your-feature-name`
3. Make your changes
4. Write/update tests if applicable
5. Run tests: `go test ./...`
6. Commit with clear messages: `git commit -m "Add feature X"`
7. Push: `git push origin feature/your-feature-name`
8. Open a Pull Request

#### Code Style
- Follow standard Go conventions (`gofmt`, `golint`)
- Keep functions small and focused
- Add comments for non-obvious logic
- Update documentation if needed

#### Commit Messages
- Use present tense: "Add feature" not "Added feature"
- Be descriptive but concise
- Reference issues: "Fix #123: Handle nil pointer in analyzer"

### Areas We Need Help

**High Priority:**
- [ ] Additional log source integrations (syslog, journald, etc.)
- [ ] More pattern matching rules for common errors
- [ ] Performance optimizations
- [ ] Documentation improvements

**Medium Priority:**
- [ ] Additional LLM provider integrations (Claude, GPT-4)
- [ ] Export formats (PDF, HTML reports)
- [ ] Custom pattern library support
- [ ] UI/UX improvements

**Nice to Have:**
- [ ] TUI mode (terminal interface)
- [ ] Plugins system
- [ ] Metrics/dashboards
- [ ] Mobile-responsive UI

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/logvoyant
cd logvoyant

# Install dependencies
go mod download

# Run locally
go run cmd/logvoyant/main.go

# Run tests
go test ./...

# Build
go build -o logvoyant cmd/logvoyant/main.go
```

## Testing

- Write tests for new features
- Ensure existing tests pass
- Add integration tests for new log sources
- Test on multiple platforms if possible (macOS, Linux)

## Documentation

- Update README.md if adding features
- Add inline comments for complex logic
- Update API documentation if changing endpoints
- Include examples for new functionality

## Questions?

- Open a [Discussion](https://github.com/SwaroopShenoy/logvoyant/discussions)
- Comment on relevant issues
- Reach out via GitHub

## Code of Conduct

Be respectful, constructive, and helpful. We're all learning here.

**Golden Rule:** Treat others how you'd want to be treated in your first open source contribution.

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

---

**Thank you for making LogVoyant better! 🚀**