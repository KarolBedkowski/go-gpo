# Changelog
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

- Supress nilaway in test; add test for apperror uniqueList
- Add more test for error wrapping & tags
