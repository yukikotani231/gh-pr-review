## Summary

- 

## Verification

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `go build ./...`

## TUI Checks

- [ ] Not a TUI change
- [ ] `go run . --fixture basic`
- [ ] Run the fixture(s) relevant to this change
- [ ] `COLUMNS=140 go run . --fixture wide-split` for split-diff-related changes

## Notes

- If this PR touches diff rendering, cursor movement, or inline threads, add or update regression tests.
