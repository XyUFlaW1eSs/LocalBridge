# Coding Style

- Run `gofmt` on all Go files.
- Keep package names short and responsibility-focused.
- Return wrapped errors with operation context.
- Pass `context.Context` through long-lived operations.
- Keep HTTP handlers thin; put domain rules in the module.
- Validate input at the boundary and cap request bodies.
- Prefer explicit structs over `map[string]any` for public payloads.
- Do not log clipboard content; clipboard data is user data.
- Use build tags for OS-specific code and keep the platform interface small.
- Add tests for public behavior and failure paths, not implementation trivia.
