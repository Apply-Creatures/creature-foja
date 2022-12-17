---
name: "Bug Report"
about: "Found something you weren't expecting? Report it here!"
title: "[BUG] "
---
<!--
NOTE: If your issue is a security concern, please email security@forgejo.org (GPG: A4676E79) instead of opening a public issue.

1. Please speak English, as this is the language all maintainers can
   speak and write.

2. Please ask questions or troubleshoot configuration/deploy problems
   in our Matrix space (https://matrix.to/#/#forgejo:matrix.org).

3. Please make sure you are using the latest release of Forgejo and
   take a moment to check that your issue hasn't been reported before.

4. Please give all relevant information below for bug reports, because
   incomplete details will be handled as an invalid report.

5. If you are using a proxy or a CDN (e.g. CloudFlare) in front of
   Forgejo, please disable the proxy/CDN fully and connect to Forgejo
   directly to confirm the issue still persists without those services.
-->

- Can you reproduce the problem on [Forgejo Next](https://next.forgejo.org/)?
- Forgejo version (or commit ref):
- Git version:
- Operating system:
- Database (use `[x]`):
  - [ ] PostgreSQL
  - [ ] MySQL
  - [ ] MSSQL
  - [ ] SQLite
- How are you running Forgejo?
<!--
Please include information on whether you built Forgejo yourself, used one of our downloads, or are using some other package.
Please also tell us how you are running Forgejo, e.g. if it is being run from docker, a command-line, systemd etc.
If you are using a package or systemd tell us what distribution you are using.
-->

## Description
<!-- Please describe the issue you are having as clearly and succinctly as possible. -->

## Reproducing
<!-- Please explain how to cause the problem to occur on demand if possible. -->

## Logs
<!--
It is really important to provide pertinent logs. We need DEBUG level logs.
Please read https://docs.gitea.io/en-us/logging-configuration/#debugging-problems
In addition, if your problem relates to git commands set `RUN_MODE=dev` at the top of `app.ini`.
Please copy and paste your logs here, with any sensitive information (e.g. API keys) removed/hidden.
-->

## Screenshots
<!-- If this issue involves the Web Interface, please provide one or more screenshots -->
