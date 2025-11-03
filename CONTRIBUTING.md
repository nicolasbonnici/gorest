# Contributing to GoREST

First off, thank you for considering contributing to GoREST! It's people like you who make GoREST such a great tool for the Go community.

## Code of Conduct

Be respectful, professional, and constructive. We're all here to build something great together.

## Quick Links

- [Report a Bug](https://github.com/nicolasbonnici/gorest/issues/new?labels=bug)
- [Request a Feature](https://github.com/nicolasbonnici/gorest/issues/new?labels=enhancement)
- [Ask a Question](https://github.com/nicolasbonnici/gorest/discussions)

## How Can I Contribute?

### 🐛 Reporting Bugs

Before creating a bug report, please check if the issue already exists. When you create a bug report, include:

- **Clear title** describing the issue
- **Steps to reproduce** the behavior
- **Expected behavior** vs actual behavior
- **Environment details**: Go version, OS, database version
- **Error messages** or logs (if applicable)

### ✨ Suggesting Features

Feature requests are welcome! Please:

- **Check existing requests** first
- **Describe the use case** (not just the solution)
- **Explain why** this would be useful to others
- **Provide examples** if possible

### 📝 Improving Documentation

Documentation improvements are always welcome:

- Fix typos or clarify existing docs
- Add examples or tutorials
- Improve code comments
- Translate documentation to other languages

### 💻 Contributing Code

#### Development Setup (5 minutes)

1. **Fork and clone** the repository
   ```bash
   git clone https://github.com/YOUR_USERNAME/gorest.git
   cd gorest
   ```

2. **Start the test database**
   ```bash
   make test-up        # Starts PostgreSQL in Docker
   make test-schema    # Loads test schema
   ```

3. **Verify your setup**
   ```bash
   make test           # All tests should pass ✅
   ```

#### Making Changes

1. **Create a branch** from `main`
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```

2. **Make your changes**
   - Write clear, readable code
   - Follow existing code style
   - Add tests for new functionality
   - Update documentation if needed

3. **Run tests**
   ```bash
   make test           # Run all tests
   make generate       # Test code generation
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

   Use [Conventional Commits](https://www.conventionalcommits.org/) format:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation
   - `test:` for tests
   - `refactor:` for code improvements
   - `chore:` for maintenance tasks

5. **Push and create a Pull Request**
   ```bash
   git push origin feature/your-feature-name
   ```

   Then open a PR on GitHub with:
   - Clear description of changes
   - Reference to any related issues
   - Screenshots/examples if applicable

## Development Guidelines

### Code Style

- Follow standard Go conventions (`gofmt`, `golint`)
- Keep functions small and focused
- Use meaningful variable names
- Add comments for complex logic

### Testing

- Write tests for new features
- Ensure all tests pass before submitting PR
- Aim for high test coverage
- Include both unit and integration tests where appropriate

### Commit Messages

Good commit messages help maintain project history:

```
feat: add MySQL support for filtering operations

- Implement filter parser for MySQL syntax
- Add tests for common filter scenarios
- Update documentation with MySQL examples

Closes #123
```

### Pull Request Process

1. **Update documentation** if you've changed functionality
2. **Add tests** for new features or bug fixes
3. **Ensure tests pass** (`make test`)
4. **Keep PRs focused** - one feature/fix per PR
5. **Be responsive** to feedback and review comments

### What We Look For

✅ **Clear purpose** - What problem does this solve?
✅ **Good tests** - Does it have adequate test coverage?
✅ **Documentation** - Is it documented for users?
✅ **Code quality** - Is it readable and maintainable?
✅ **No breaking changes** - Or are they well-justified and documented?

## Project Structure

```
gorest/
├── cmd/                  # CLI tools (modelgen, resourcegen, openapigen)
├── pkg/gorest/           # API server entrypoint
├── pkg/database/         # Database abstraction layer
├── internal/             # Core logic
│   ├── models/          # Generated database models
│   ├── api/             # Generated API code
│   │   ├── dtos/        # Data Transfer Objects
│   │   └── resources/   # REST endpoints
│   ├── crud/            # Generic CRUD operations
│   ├── hooks/           # Business logic hooks
│   └── helpers/         # HTTP helpers and middleware
├── test/                # Test utilities and schemas
└── Makefile            # Build and test commands
```

## Good First Issues

Looking for a place to start? Check out issues labeled [`good first issue`](https://github.com/nicolasbonnici/gorest/labels/good%20first%20issue).

Some ideas for first contributions:

- Add examples to HOOKS.md
- Improve error messages
- Add tests for edge cases
- Fix typos in documentation
- Add code comments

## Getting Help

- 💬 **GitHub Discussions** - Ask questions
- 🐛 **GitHub Issues** - Report bugs or request features
- 📧 **Email maintainers** - For sensitive matters

## Recognition

All contributors will be:

- Listed in our README
- Acknowledged in release notes
- Given credit in commit history

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to GoREST!** 🚀

Every contribution, no matter how small, helps make GoREST better for everyone.
