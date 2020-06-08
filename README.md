[![CircleCI](https://circleci.com/gh/Dynom/ERI.svg?style=svg)](https://circleci.com/gh/Dynom/ERI)
[![Go Report Card](https://goreportcard.com/badge/github.com/Dynom/ERI)](https://goreportcard.com/report/github.com/Dynom/ERI)
[![GoDoc](https://godoc.org/github.com/Dynom/ERI?status.svg)](https://pkg.go.dev/github.com/Dynom/ERI)
[![codecov](https://codecov.io/gh/Dynom/ERI/branch/master/graph/badge.svg)](https://codecov.io/gh/Dynom/ERI)


# ERI
Email Recipient Inspector is a project for preventing email typos. It's a self-learning service, or a command line utility, which you can employ to help users prevent mistakes when entering their email address.

# ERI as command line utility
## Installation
Either download the binaries, or follow the typical Go installation process. While all examples here use the bash shell, it should work on Windows as well.
```bash
$ go install github.com/Dynom/ERI/cmd/eri-cli
$ eri-cli -h
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
## Basic usage
Running
```bash
$ eri-cli check --input-is-domain gmail.com | jq .
```
Produces
```json
{
  "input": "gmail.com",
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
## With reporting
```bash
$ bzcat domains.bz2 | \
  eri-cli check --input-is-domain --resolver 8.8.8.8 | \
  eri-cli report --details stats
{"passed":32343,"rejected":1293,"run_duration_ms":1381800}
```


# ERI as web service
## Endpoints
Each request must be accompanied by a `Content-Type: application/json` header. Other than basic JSON, ERI also supports [GraphQL](https://graphql.org/). 

### /suggest
The Suggestion endpoint returns a list of 1 or more equally good, or better alternatives. When no better match has been found, the input will be returned. The `malformed_syntax` field is a boolean indicating whether the input is never valid (true), or _might_ be (false). This is intentionally vague, since it's impossible to know if an email address can be considered [legitimate](#email-delivery-nuances).

```bash
curl -s 'http://localhost:1338/suggest' \
  -H 'Content-Type: application/json' \
  -d '{"email": "john.doe@example.rg"}'
```
#### Request
```json
{
  "email": "john.doe@example.rg"
}
```

#### Response
The local part (left of the `@`) remains completely untouched. It's simply echoed back from the input.
```json
{
  "alternatives": [
    "john.doe@example.org"
  ],
  "malformed_syntax": false
}
```

### /autocomplete
The autocomplete endpoint returns a list of domains matching the prefix. To prevent leaking sensitive information, ERI is configured with a threshold to limit exposure of rarely used domains.
```bash
curl -s 'http://localhost:1338/autocomplete' \
  -H 'Content-Type: application/json' \
  -d '{"domain": "g"}'
```
#### Request
```json
{
  "domain": "g"
}
```
#### Response
```json
{
  "suggestions": [
    "gmail.com"
  ]
}
```

# ERI as a library
```bash
$ go get -u github.com/Dynom/ERI
```

# ERI design goals
## Fast
It uses an incremental approach to determining correctness: Syntax, DNS and optionally more

## Privacy by design
It employs several [configuration options](https://github.com/Dynom/ERI/blob/master/cmd/web/config.toml) to limit exposure, and it only keeps an obfuscated local part in memory.

## Scales pretty well
Depending on the setup, each instance can handle hundreds of requests per second, and it coordinates its state with multiple instances.

# ERIs Learning
Certain typos lead to unintended but "correct" domains. One example is: `hotmai.com` versus `hotmail.com`. An easy typo to make, but harder to distinguish what the user intended (since `hotmai.com` is a valid domain).

To solve this ERI learns from both good and bad results, to form a bias for the more likely domain that is intended. This bias is specific to a workload. ERI offers two endpoints to help the user identify a mistake. Auto completion and Alternatives Suggestions.

# Suggestions
ERI uses [TySug](https://github.com/Dynom/TySug) to help with finding alternatives and supports various algorithms for fuzzy matching

# ERI versus Email validation
ERI is a service which is designed to help in legitimate use-cases to prevent mistakes. [It doesn't claim correctness as you might expect](#email-delivery-nuances), but it will offer useful hints to a user that something might be wrong, even when the syntax is actually correct.

# Installation
## Server
Download a binary and take it for a spin. The default configuration should get you up-and-started in pretty quick.
```bash
./eri -backend-driver=memory -listen-on="localhost:1338"
```  
## Client
```js
@todo

```

# ERIs design
## Multiple instances
ERI communicates by a broadcasting setup. Currently, GCPs pub/sub and Postgres listen/notify is on the wishlist. This is chatty with many instances, however for a small setup, handling up to 10.000 req/s, this works quite well.

## Persistence
ERI uses Postgres as persistence backend.

## Releases
ERI currently follows the semver notation, this will probably change in the future.

ERI tries to stay current with Go's version releases, it might not build on older versions. But it will very likely build on a recent version.

The `master` branch is fairly stable. Most work is done in feature-branches. 

## Comparing to alternatives
### Various inbox-check-services
A quick search will give you many hits for services that validate your list of emails for you. Often with delivery guarantees. This forces you to give away your users-data to them. With ERI you can get pretty close, without needing to do that.

These services will give you a lot more functionality though.

### Mailcheck.js
Mailcheck works completely in JavaScript, with options to white-list TLDs, domains etc. It differs from ERI in that ERI runs server-side, and it takes a self-learning approach, based on your existing users.


# Email delivery nuances
Ever since the first e-mail got sent in 1971 a lot has happened with electronic mail. In modern days email is seen as "the" way to identify and communicate with people online. Because of this, many people will easily give away their email addresses and people receive many, many emails. It's hard to read it all, not even counting the spam. Looking specifically at my own behaviour, I don't even open email unless I think it's important, just by scanning the sender and the subject of the email.

With this in mind, even with a perfect validator, and a brilliantly composed and relevant email, it's still possible your email won't be read. ERI is designed to help out the user willing to trust you with their email address. ERI is not designed as a marketing tool to help optimise email delivery.

# Security and Privacy
ERI is consciously made with Security and Privacy in mind. If you find something that could be improved, let me know! Feel free to file an issue or email me directly.

## Disclosure
Please contact me at mark@dynom.nl before disclosing publicly.