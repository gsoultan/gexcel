# gexcel — task completion checklist

Before a task is done:

1. `gofmt -l .` prints nothing; `go vet ./...` clean; `go test -race ./...` green.
2. `./scripts/check-headers.sh` passes (every `.go` file starts with the copyright line + blank line).
3. New behaviour has a white-box test next to the code and, when user-visible, a black-box test in
   `tests/` or `gexcel_test.go`. Parsers get a fuzz seed.
4. When output changes, run `GEXCEL_EXTERNAL=1 go test -run External ./tests/` locally (needs
   `soffice` and `openpyxl`) or check the `external` CI job.
5. Update `.serena/memories/feature_coverage.md` when a feature's status changes, and the other
   memories when the architecture or conventions change.
6. Run `graphify update .` so `graphify-out/` matches the code; commit it with the change.
7. Commit as `gsoultan <gembit.shirazi@gmail.com>`. No Co-Authored-By, session links or AI credit
   lines in commits, PR bodies, comments or reviews. After pushing, check the PR on GitHub for any
   automatically added credit lines and remove them.
8. PR bodies start with the project attribution block provided by the project instructions.
