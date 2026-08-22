# Jigyll

[![go badge][go-svg]][go-url]
[![Golangci-lint badge][golangci-lint-svg]][golangci-lint-url]
[![Coveralls badge][coveralls-svg]][coveralls-url]
[![Go Report Card badge][go-report-card-svg]][go-report-card-url]


Jigyll is a partially-compatible clone of the [Jekyll](https://jekyllrb.com)
static site generator, written in the [Go](https://golang.org) programming
language. It provides `build` and `serve` commands, with directory watch and
live reload.


## [Optional] Install command-line autocompletion

Add this to your `.bashrc` or `.zshrc`:

```bash
# Bash:
eval "$(jigyll --completion-script-bash)"
# Zsh:
eval "$(jigyll --completion-script-zsh)"
```

## Development

This project uses [just](https://github.com/casey/just) as a command runner. Run `just` to see available recipes:

```bash
just build      # compile the binary
just buildlinux # cross-compile for linux (amd64 + arm64)
just clean      # remove build artifacts
just install    # install the binary
just lint       # run linter
just release    # bump patch version, tag, and push
just test       # run tests
```

## Upstream credit

Jigyll is a fork of [gojekyll](https://github.com/osteele/gojekyll). We gratefully credit its principal contributors: [Oliver Steele](https://github.com/osteele), [Bjørn Erik Pedersen](https://github.com/bep), [Maurits van der Schee](https://github.com/mevdschee), and [Daniil Gentili](https://github.com/danog).

[coveralls-url]: https://coveralls.io/r/reidransom/jigyll
[coveralls-svg]: https://img.shields.io/coveralls/reidransom/jigyll.svg?branch=main
[license-url]: https://github.com/reidransom/jigyll/blob/main/LICENSE
[license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
[go-url]: https://github.com/reidransom/jigyll/actions?query=workflow%3A%22Build+Status%22
[go-svg]: https://github.com/reidransom/jigyll/actions/workflows/go.yml/badge.svg
[golangci-lint-url]: https://github.com/reidransom/jigyll/actions?query=workflow%3Agolangci-lint
[golangci-lint-svg]: https://github.com/reidransom/jigyll/actions/workflows/golangci-lint.yml/badge.svg
[go-report-card-url]: https://goreportcard.com/report/github.com/reidransom/jigyll
[go-report-card-svg]: https://goreportcard.com/badge/github.com/reidransom/jigyll
