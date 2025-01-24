## Running Tests and Checking Coverage in Go

To ensure your Go project is functioning correctly and meeting code quality standards, follow the steps below to run tests and check code coverage.

### Running Tests

You can run all tests in the project by executing the following command:

```bash
make test
```

### Generating a Coverage Report

To generate a detailed coverage report and save it to a file:

```bash
go test -coverprofile=coverage.out ./...
```

### Viewing Coverage Report in Terminal

Once the coverage profile is generated, display function-wise coverage:

```bash
go tool cover -func=coverage.out
```

### Viewing Coverage in HTML Format

To visualize code coverage in a browser:

```bash
go tool cover -html=coverage.out
```

