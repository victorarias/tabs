# Tabs Test - Ralph Loop

This is a test run to verify the Ralph loop works correctly.

## Your Task

1. Read `prd-test.json` to see the test story
2. Read `progress-test.txt` to see progress
3. Find the story where `passes: false`
4. Implement it:
   - Create internal/hello/hello.go
   - Add package hello
   - Add function: `func Greet(name string) string` that returns "Hello, {name}!"
   - Keep it simple and clean
5. Run quality checks:
   - `make build` - must succeed
   - `go vet ./...` - must pass
6. If checks pass:
   - Commit with message: "Add hello package"
   - Update `prd-test.json`: set `passes: true` for the story
   - Append to `progress-test.txt`: "✓ Added hello package with Greet function"
7. If checks fail, fix and try again

## Expected Result

File `internal/hello/hello.go`:
```go
package hello

func Greet(name string) string {
    return "Hello, " + name + "!"
}
```

That's it! Simple test to verify the loop works.
