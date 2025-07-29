test:
	go test -race -v ./ || exit 1;

tidy:
	go mod tidy

coverage:
	go test $$(go list ./... | grep -v /mocks) -coverprofile=coverage.out
	go tool cover -func=coverage.out
