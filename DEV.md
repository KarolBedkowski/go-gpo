DEV help
========

Tests
-----

### Database used for tests

~~~~ shell
export GOGPO_TEST_DB_DRIVER=postgres
export GOGPO_TEST_DB_CONNSTR='host=127.0.0.1 user=gogpo password=gogpo123 database=gogpo_tst'
~~~~

or (default):

~~~~ shell
export GOGPO_TEST_DB_DRIVER=sqlite
export GOGPO_TEST_DB_CONNSTR=':memory:'
~~~~


Debug flags
-----------

List separated by comma.

 -  Env: `GOGPO_DEBUG`
 -  Flag: `--debug`

Values:

 -  `logbody` - enable logging request and response body and headers.
 -  `do` - enable logging samber/do and /debug/do endpoint.
 -  `go` - enable /debug/pprof endpoint.
 -  `router` - show defined routes.
 -  `querymetrics` - enable metrics for query metrics
 -  `flightrecorder` -  enable flight recorder for long server queries (threshold
    defined by `GO_GPO_DEBUG_FLIGHTRECORDER_THRESHOLD` environment variable;
    default 1s)
 -  `trace` - enable tracing with net/trace, x/net/trace (requests and events)
 -  `all` enable all debug flags.
