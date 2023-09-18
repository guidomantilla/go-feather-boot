.PHONY: phony
phony-goal: ; @echo $@

validate: generate sort-import format vet lint coverage

generate:
	go generate ./pkg/... ./tools/...

sort-import:
	goimports-reviser -rm-unused -set-alias -format -recursive pkg

format:
	go fmt ./pkg/...

vet:
	go vet ./pkg/...

lint:
	golangci-lint run ./pkg/...

test:
	go test -covermode count -coverprofile coverage.out.tmp.01 ./pkg/...
	cat coverage.out.tmp.01 | grep -v "mocks.go" > coverage.out

coverage: test
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html

graph:
	godepgraph -s . | dot -Tpng -o godepgraph.png

sonarqube: coverage
	sonar-scanner

update-dependencies:
	go get -u ./...
	go get -t -u ./...
	go mod tidy

prepare:
	go install github.com/incu6us/goimports-reviser/v3@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang/mock/mockgen@latest
	go install github.com/cweill/gotests/gotests@latest
	go mod download
	go mod tidy