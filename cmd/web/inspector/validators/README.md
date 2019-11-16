Rethinking Validation.

An e-mail address check typically flows from least to most expensive:
1. Basic syntax
1. Domain DNS lookup 
1. Domain MX lookup
1. MX connect
1. MX [Mail &] xRCPT command

Some of these checks can be cached and when they have been performed before, it makes less sense to do it again. An e-mail address consists of: local (left of the @) and the domain part, it makes sense to check and cache the checks of the domain separately. Since the DNS, MX and MX connect steps are domain specific. If these produce a result for john@example.org, they will be the same for jane@example.org. Especially on domains with many e-mail addresses (e.g. gmail.com) this is a worthy investment.


Validators should test small parts of this validation chain:

1. Validate syntax
2. Domain MX lookup
3. MX DNS lookup
4. MX Connect ports 25, 587, 2525 or 465
5. Mail commands (Mail & RCPT) 

Validators can reuse artifacts produced by previous validators (e.g. MX lookup) and they have a significant order (validating the syntax last makes little sense).

A limited state-machine makes sense:




