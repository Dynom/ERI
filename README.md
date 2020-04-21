[![CircleCI](https://circleci.com/gh/Dynom/ERI.svg?style=svg)](https://circleci.com/gh/Dynom/ERI)
[![Go Report Card](https://goreportcard.com/badge/github.com/Dynom/ERI)](https://goreportcard.com/report/github.com/Dynom/ERI)
[![GoDoc](https://godoc.org/github.com/Dynom/ERI?status.svg)](https://pkg.go.dev/github.com/Dynom/ERI)
[![codecov](https://codecov.io/gh/Dynom/ERI/branch/master/graph/badge.svg)](https://codecov.io/gh/Dynom/ERI)


# ERI
Email Recipient Inspector is a project for preventing e-mail typos. It's a self-learning service, which you can employ to help users prevent mistakes when entering their e-mail address.

## Endpoints
Each request must be accompanied by a `Content-Type: application/json` header. Besides plain JSON, ERI also supports [GraphQL](https://graphql.org/). 

### /suggest
The Suggestion endpoint returns a list of 1 or more equally good, or better alternatives. If no better match is found, the input is returned. The `malformed_syntax` field is a boolean indicating whether the input is never valid, or _might_ be.

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
```json
{
  "alternatives": [
    "john.doe@example.org"
  ],
  "malformed_syntax": false
}
```

### /autocomplete
The autocomplete endpoint returns a list of domains matching the prefix.
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


# ERI design goals
## Fast
It uses an incremental approach to determining correctness: Syntax, DNS and optionally more

## Privacy by design
It employs several configuration options to limit exposure, and it only keeps an obfuscated local part in memory.

## Scales pretty well
Depending on the setup, each instance can handle hundreds of requests per second, and it coordinates its state with multiple instances.

# ERIs Learning
Certain typos lead to unintended but "correct" domains. One example is: `hotmai.com` versus `hotmail.com`. An easy typo to make, but harder to distinguish what the user intended (since `hotmai.com` is a valid domain).

To solve this ERI learns from both good and bad results, to form a bias for the more likely domain that is intended. This bias is specific to a workload.

# Suggestions
ERI uses [TySug](https://github.com/Dynom/TySug) to help with finding alternatives and supports various algorithms for fuzzy matching

# ERI versus E-mail validation
ERI is a service which is designed to help in legitimate use-cases to prevent mistakes. It doesn't claim correctness, but it will offer useful hints to a user that something might be wrong, even when the syntax is actually correct.

It's also not designed as a Marketing tool to help in optimising a contact list

# ERIs design
## Multiple instances
ERI communicates by a broadcasting setup. Currently, GCP's pub/sub and Postgres listen/notify is on the wishlist. This is chatty with many instances, however for a small setup, handling up to 10.000 req/s, this works quite well.

## Persistence
ERI uses Postgres as persistence backend.

## Releases
ERI currently follows the semver notation, this will probably change in the future.

ERI tries to stay current with Go's version releases, it might not build on older versions. But it will very likely build on a recent version.

The `master` branch is fairly stable. Most work is done in feature-branches. 

# Security Disclosure
Please contact me at mark@dynom.nl before disclosing publicly.