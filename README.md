# ERI
Email Recipient Inspector, checks e-mail addresses and offers suggestions when there is suspicion of a typo. It uses an incremental validation approach and support learning

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
    - Should support Unicode domains
- Learn from validations
    - Skip MX lookups when a domain is known to be good
    - Skip expensive validation when an e-mail address has been "recently" validated
    - Reject early when a recent validation resulted in an error

# Features
## Implemented
## Planned
- Play nice with Mail providers webhooks (e.g.: AWS SES/SNS, Send in Blue, Mailgun) for learning of failure(s)
- 


# Considerations and Design Choices
## Validation
ERI performs an incremental validation approach. It starts with "cheap" checks and can continue until issuing MAIL 
commands. Please continue reading on learning why the latter isn't recommended in all situations.

ERI supports including MAIL commands, but for typical real world usage this is a bad idea. Large mail providers (e.g.: gmail or hotmail) ignore/block these commands for security reasons. Secondly there is a semantical argument to not chose this option, in that you have to consider what you're trying to accomplish. No e-mail validation service can guarantee that an e-mail is received, doesn't end up in spam and is actually read by the recipient.

For specific situations, however, you can use ERI to perform these lookups. E.g. for internal services where you want to validate (old) addresses.

## Data storage
//todo

## Control
- web service
- CLI to manage the service
- Database to persist state
- Pub/Sub to coordinate between multiple services


## Existing but unintended domain
Certain typos lead to unintended but "correct" domains. One example is: hotmai.com versus hotmail.com. An easy typo to 
make, but harder to distinguish what the user intended.

For this and other situations ERI learns about e-mail addresses. See the section Learning

# ERI's Learning
ERI does some additional bookkeeping when a check is performed or when an explicit learn call is made. This process of 
learning is useful to form a bias in favor of more "likely to be correct" addresses.

## Addresses versus domains
ERI learn's about domains (the part after the @) and the corresponding local parts (the part before the @). The local part
is used to help determine the usage frequency, which helps create a favorability towards more common domains.