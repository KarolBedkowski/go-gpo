# Changelog

## v1.1.3 2025-12-28

- Fix broken save settings


## v1.1.2 2025-12-27

### Bug Fixes

- web: Layout
- Fix types in settings table
- Email is required but can be not unique
- Fix migration with disabled fk
- Fix default connection string (database argument)
- Fix onclose/onconnect script; add PRAGMA busy_timeout
- Logging objects
- Fix updating podcasts info; load only changed episodes/podcasts
- Do not remove / suffix from podcasts urls
- When serach for podcast by url - try match also url without / suffix
- Fix log podcast downloader
- Normalize podcast url; similar to mygpo
- Sanitize episode url when loading podcast data
- Fix handle podcast-load-interval parameter
- Do not aggregate episodes when show last actions
- Serialize json without temporary buffer
- Add user to request log; rename uri->url
- Check is stderr is console output when configure logging
- Proper formatting logfmt log format
- Quote error value in logfmt log format
- Better loging in podcast downloader
- Drop "text" log format; autodetect format by default

### Features

- Download episode info load also episodes title and guid
- Show episodes title if available
- Shorten long podcasts description
- Allow set load podcasts interval by cli argument
- Podcast downloader skip not modified feeds
- Downloading episodes data is disable by default



## v1.1.1 2025-12-16

### Bug Fixes

- Sanitize url remove leading / from urls
- Fix replace/update subscription when add already subscribed podcast
- Make sure that update_at is valid before save for dev/podcast/episodes
- Don't filter podcasts/episodes by update_at when not needed
- Datafix for broken podcasts.updated_at
- Do not require device on change subscription; fix resubscribe

### Miscellaneous Tasks

- Fix typo


## v1.1.0 2025-12-15

### Bug Fixes

- Fix & improve web pages
- Add unique index on user podcasts url; make transactions serializable
- Fix upload action for unsubscribed podcasts, new devices
- All errors other than validation error are serious

### Features

- Better validation for user/devices names
- Add/fix more input data validators
- Podcast can be deleted

### Miscellaneous Tasks

- Update .air.toml
- Update deps
- Update makefile

### Refactor

- Migrate to quicktemplate
- Add pagecontext & renderer service for web pages
- Migrate to quicktemplate

### Testing

- Suppress nilaway in test; add test for apperror uniqueList
- Add more test for error wrapping & tags
