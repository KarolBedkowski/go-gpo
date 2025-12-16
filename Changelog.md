# Changelog

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
