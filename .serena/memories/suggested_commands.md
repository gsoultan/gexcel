# gexcel — suggested commands

```
go build ./...                                   # compile everything
gofmt -l .                                       # must be empty
go vet ./...
go test -race ./...                              # unit + black-box tests
go test -run=^$ -fuzz=FuzzParse -fuzztime=30s ./formula
go test -run=^$ -fuzz=FuzzRead  -fuzztime=30s ./reader
GEXCEL_EXTERNAL=1 go test -run External ./tests/ # LibreOffice + openpyxl validation
GEXCEL_OUT=/tmp/out go test ./writer .           # writes sample.xlsx / stream.xlsx / api.xlsx
go run ./cmd/gexcel-dump file.xlsx               # dump what the reader sees
./scripts/check-headers.sh                       # copyright header check
graphify update .                                # refresh graphify-out/ (run before committing)
soffice --headless --convert-to csv --outdir out file.xlsx   # let LibreOffice evaluate formulas
```
