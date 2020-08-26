go-fmt:
	@echo -n Checking format...
	@test $(shell goimports -l ./ | grep -v mock | wc -l) = 0 \
		|| { echo; echo "some files are not properly formatted";\
		echo $(shell goimports -l ./ | grep -v mock);\
		exit 1;}\

	@echo " everything formatted properly"	

go-lint:
	@echo -n Checking with linter...
	@test $(shell golint ./... | wc -l) = 0 \
		|| { echo; echo "some files are not properly linted";\
		echo $(shell golint ./...);\
		exit 1;}\

	@echo " everything linted properly"	

cover-report:
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out -o coverage-summary.txt

test:
	@echo "Testing ..."
	@go test -coverprofile=coverage.out -failfast ./...

test-with-report: test cover-report

code-up-to-date: go-fmt go-lint

all: code-up-to-date test-with-report 
