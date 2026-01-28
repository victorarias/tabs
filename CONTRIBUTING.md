# Contributing to tabs

Thank you for your interest in contributing to **tabs**! We welcome contributions from the community.

## Code of Conduct

Be respectful, constructive, and collaborative. We're all here to learn and share knowledge.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/yourorg/tabs/issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Your environment (OS, Go version, etc.)
   - Relevant logs or screenshots

### Suggesting Features

1. Check [Discussions](https://github.com/yourorg/tabs/discussions) for similar ideas
2. Create a new discussion in "Ideas" category
3. Describe the problem you're trying to solve
4. Propose your solution
5. Discuss tradeoffs and alternatives

### Pull Requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Write or update tests
5. Run tests: `make test`
6. Commit with clear messages: `git commit -m "Add feature: my feature"`
7. Push to your fork: `git push origin feature/my-feature`
8. Open a Pull Request

## Development Setup

See [README.md#development](README.md#development) for setup instructions.

## Testing

- Write tests for new features
- Ensure all tests pass before submitting PR
- Run: `make test`

## Documentation

- Update relevant docs in `docs/` if you change architecture or APIs
- Update README.md if you add user-facing features
- Use clear, concise language

## Commit Messages

Follow conventional commits:
- `feat: add cursor support`
- `fix: resolve daemon race condition`
- `docs: update architecture diagram`
- `test: add unit tests for JSONL parser`
- `refactor: simplify socket protocol`

## License

By contributing, you agree that your contributions will be licensed under AGPLv3.
