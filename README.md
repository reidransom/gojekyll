# Jigyll


[![go badge][go-svg]][go-url]
[![Golangci-lint badge][golangci-lint-svg]][golangci-lint-url]
[![Coveralls badge][coveralls-svg]][coveralls-url]
[![Go Report Card badge][go-report-card-svg]][go-report-card-url]


Jigyll is a partially-compatible clone of the [Jekyll](https://jekyllrb.com)
static site generator, written in the [Go](https://golang.org) programming
language. It provides `build` and `serve` commands, with directory watch and
live reload.

| &nbsp;                  | Jigyll                                  | Jekyll | Hugo                         |
| ----------------------- | ----------------------------------------- | ------ | ---------------------------- |
| Stable                  |                                           | ✓      | ✓                            |
| Fast                    | ✓<br>([~20×Jekyll](./docs/_docs/benchmarks.md)) |        | ✓                            |
| Template language       | Liquid                                    | Liquid | Go, Ace and Amber templates  |
| SASS                    | ✓                                         | ✓      | ✓                            |
| Jekyll compatibility    | [partial](./docs/_docs/differences.md)    | ✓      |                              |
| Plugins                 | [some](./docs/_docs/plugins.md)           | yes    | shortcodes, theme components |
| Windows support         | ✓                                         | ✓      | ✓                            |
| Implementation language | Go                                        | Ruby   | Go                           |

<!-- TOC -->

- [[Optional] Install command-line autocompletion](#optional-install-command-line-autocompletion)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Upstream credit](#upstream-credit)
- [Attribution](#attribution)
- [Related](#related)
- [License](#license)

<!-- /TOC -->

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

## Troubleshooting

If the error is "403 API rate limit exceeded", you are probably building a
repository that uses the `jekyll-github-metadata` gem. Try setting the
`JEKYLL_GITHUB_TOKEN`, `JEKYLL_GITHUB_TOKEN`, or `OCTOKIT_ACCESS_TOKEN`
environment variable to the value of a [GitHub personal access
token][personal-access-token] and trying again.

[personal-access-token]: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token

## Upstream credit

Jigyll is a fork of [gojekyll](https://github.com/osteele/gojekyll). We gratefully credit its principal contributors: [Oliver Steele](https://github.com/osteele), [Bjørn Erik Pedersen](https://github.com/bep), [Maurits van der Schee](https://github.com/mevdschee), and [Daniil Gentili](https://github.com/danog).

## Attribution

In addition to being totally and obviously inspired by Jekyll and its plugins,
Jekyll's solid _documentation_ was indispensible --- especially since I wanted
to implement Jekyll as documented, not port its source code. The [Jekyll
docs](https://jekyllrb.com/docs/home/) were always open in at least one tab
during development.


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
