
# Go logging spike 2017-09-26

## Requirements

1. Leveled logging, ideally trace, debug, info, error.
2. Configuration driven log format.
3. Sane API (hard to define).
4. Speed.

## Comparison

[Dave Cheney][logging] makes some interesting arguments about logging
simplicity. Instead of a proliferation of levels he argues for fine-grained
control of where to disable/enable logging with a package or even finer scope.

The [loggly] article does not really offer anything to the discussion, besides
some links: [libs.club] appears to be the most comprehensive list of logging
packages for go.

This [reddit] thread is comprehensive as well, adding a bit of background and
opinions to every framework discussed. Perhaps it's worth looking into a
combined logging and metrics package like [health], though that particular one
looks unmaintained. The majority seem to favour "structured logging", as
demonstrated by [logrus], [log15] and [go-kit/log]. Not sure I'm convinced by
this approach: I can see distinct advantages for log indexing and query
purposes, but following logs in output and grepping seems somewhat harder, and
the API is uglier. The [go-kit] package seems worth looking into, as it appears
to bundle a lot of useful components, kinda like dropwizard.

[go-logging] appears to be simple, and on the surface meets the requirements. It
relies on global log objects. Configuration is reminiscent of the python logging
module. Multiple backends. It does however appear unmaintained (last commit 2
years ago).

[zap] claims to be really fast and is actively maintained. It has a
non-standard, slightly bizarre API, under-explored in the README, so this needs
a follow-up with some code. It's also unclear how it is configured.

[glog] is the one used by geth iirc, supports custom levels, but API is
weird/ugly.

[loggo] looks somewhat unmaintained (7 months ago).

## Links

[go-logging]: https://github.com/op/go-logging
[logrus]: https://github.com/sirupsen/logrus
[zap]: https://github.com/uber-go/zap
[glog]: https://godoc.org/github.com/golang/glog
[loggo]: https://github.com/juju/loggo

[loggly]: https://www.loggly.com/blog/think-differently-about-what-to-log-in-go-best-practices-examined/
[reddit]: https://www.reddit.com/r/golang/comments/3pshhg/whats_your_favorite_log_library/
[logging]: https://dave.cheney.net/2015/11/05/lets-talk-about-logging
[libs.club]: http://libs.club/golang/developement/logging

[log15]: https://github.com/inconshreveable/log15
[go-kit/log]: https://github.com/go-kit/kit/tree/master/log
[go-kit]: https://github.com/go-kit/kit
[health]: https://github.com/gocraft/health
