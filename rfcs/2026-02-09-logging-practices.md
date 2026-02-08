+++
title = "Logging"
author = "cheongyx@cardiff.ac.uk"
date = "2026-02-09"
+++

## Summary

We're using [zerolog](https://github.com/rs/zerolog). It's performant and allows us to easily control the log level
depending on whether we're debugging an issue or just checking for metrics. It also allows logging in JSON, which allows
tools like Promtail/Loki/Grafana to easily parse and aggregate metrics. This is useful in production to detect elevated
error rates and whatnot.

We wrap zerolog in `pkg/logger/logger.go` so that we have a more convenient interface that is also generic to make it
easy to swap out the logging implementation when necessary.

There are 6 levels of logging, from most to least verbose

- **Trace** \- Only when I would be "tracing" the code and trying to find one **part** of a function specifically.
- **Debug** \- Information that is diagnostically helpful to people more than just developers (IT, sysadmins, etc.).
- **Info** \- Generally useful information to log (service start/stop, configuration assumptions, etc). Info I want to
  always have available but usually don't care about under normal circumstances. This is my out-of-the-box config level.
- **Warn** \- Anything that can potentially cause application oddities, but for which I am automatically recovering.
  (Such as switching from a primary to backup server, retrying an operation, missing secondary data, etc.)
- **Error** \- Any error which is fatal to the **operation**, but not the service or application (can't open a required
  file, missing data, etc.). These errors will force user (administrator, or direct user) intervention. These are
  usually reserved (in my apps) for incorrect connection strings, missing services, etc.
- **Fatal** \- Any error that is forcing a shutdown of the service or application to prevent data loss (or further data
  loss). I reserve these only for the most heinous errors and situations where there is guaranteed to have been data
  corruption or loss.

Credits: <https://stackoverflow.com/a/2031209>
