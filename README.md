# ERI
Email Recipient Inspector, checks e-mail addresses and offers suggestions when there is suspicion of a typo.

ERI design goals:
- Fast
    - Incremental expensiveness on checks syntax -> MX -> RCPT
- Secure
    - Fuzzer tested
    - Rate limited
- Privacy conscious
    - Tries to hash the in-memory result to leak as little as possible on bugs / compromise
- Reliable
    - Support temporary errors
    - Learnings should be persisted
    - Services should be stateless
    - Services should coordinate learnings / cache
- Learn from validations
    - Skip MX lookups when a domain is known to be good
    - Skip expensive validation when an e-mail address has been "recently" validated
    - Reject early when a recent validation resulted in an error
     


POST /learn
X-Authentication: ..

    // One-off's, detailed
    {"domain": "grr.la",          "validations": <mask>}
    {"email":  "john.doe@grr.la", "validations": <mask>}

    {"email":  "jack@example.org", "validations": <mask>}
    
    // Bulk, not-so-detailed
    {
        "emails":[
            {"value":"foo@gmail.com", "valid": true},
            {"value":"bar@gmail.com", "valid": true},
            {"value":"baz@gmail.con", "valid": false}
        ],
        "domains": [
            {"value": "gmail.com", "valid": true},
            {"value": "tysug.net", "valid": true},
            
            {"value": "gmail.con",   "valid": false},
            {"value": "hotmail.con", "valid": false},
        ]
    }

Responses:
    {"status": "OK", "total_entries": 123124, ...}


POST /check

    {"domain": "grr.la"}
    {"email":  "john.doe@grr.la"}
    {"email":  "john.doe@grr.la", "with_alternatives": true}

Responses:
    {"valid": true }
    {"valid": false, "alternative": "john.doe@gmail.com"}


POST /alternatives
    {"email": "john.doe@gmail.con", "limit": 5}

Responses:
    {"alternatives": [{"email":"john.doe@gmail.com", "score": 0.96666}]}