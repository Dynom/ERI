# ERI's CLI
```
eri-cli -h
CLI Compagnion of ERI

Usage:
  eri-cli [command]

Available Commands:
  check       Validate email addresses
  help        Help about any command
  report      Reporting companion to check

Flags:
  -h, --help   help for eri-cli

Use "eri-cli [command] --help" for more information about a command.
```
## Installation
### With Go installed
```bash
go get -u github.com/Dynom/ERI/cmd/eri-cli
```

### Download a release
Download from: https://github.com/Dynom/ERI/releases

# Commands
## check
Validates one or more e-mail addresses.

### Examples
#### A basic run
```bash
$ eri-cli check john.doe@example.org
```
```json
{
  "input": "john.doe@example.org",
  "valid": true,
  "checks_run": [
    "syntax",
    "lookup",
    "domainHasIP"
  ],
  "checks_passed": [
    "syntax",
    "lookup",
    "domainHasIP"
  ],
  "version": 2
}
```

Notes:
- This result is actually incorrect and only serves as an example.
- The version field, is for discriminating on future changes. Any change to the structure of the output will change the version number to a new unique value.

#### Processing a list
```bash
$ cat emails.csv | eri-cli check > result.json
```

Directly from a database
```bash
$ echo "copy (select email from users) to STDOUT WITH CSV" | \
    psql <connection details> | \
    eri-cli check --resolver 1.1.1.1 | \
    tee result.json | \
    eri-cli report --only-invalid
```

Piping through compressed
```bash
bzcat emails.bz2 | \
    eri-cli check | \
    eri-cli report --only-invalid > invalid-emails.json
```

## report
Produces a report from a check run or previously recorded run and adds more power when it comes to reporting styles

### Examples
Reading from a previous run
```bash
bzcat report.bz2 | eri-cli report
```

Pipe invalid results through to an external program
```bash
bzcat emails.bz2 | \
    eri-cli check --resolver 1.1.1.1 | \
    eri-cli report --only-invalid | \
    jq .email | \
    xargs ./updateStatus.sh 
```
