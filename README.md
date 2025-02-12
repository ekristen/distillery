# Distillery

![GitHub License](https://img.shields.io/github/license/ekristen/distillery)
[![Known Vulnerabilities](https://snyk.io/test/github/ekristen/distillery/badge.svg)](https://snyk.io/test/github/ekristen/distillery)
[![Go Report Card](https://goreportcard.com/badge/github.com/ekristen/distillery)](https://goreportcard.com/report/github.com/ekristen/distillery)
![GitHub Release](https://img.shields.io/github/v/release/ekristen/distillery?include_prereleases)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/ekristen/distillery/total)

## Overview

Without a doubt, [homebrew](https://brew.sh) has had a major impact on the macOS and even the linux ecosystem. It has made it easy
to install software and keep it up to date. However, it has been around for 15+ years and while it has evolved over time,
its core technology really hasn't changed, and 15+ years is an eternity in the tech world. Languages like [Go](https://golang.org)
and [Rust](https://www.rust-lang.org) have made it easy to compile binaries and distribute them without complicated
installers or dependencies. **I love homebrew**, but I think there's room for another tool.

**dist**illery is a tool that is designed to make it easy to install binaries on your system from multiple different
sources. It is designed to be simple and easy to use. It is **NOT** designed to be a package manager or handle complex
dependencies, that's where homebrew shines.

The goal of this project is to install binaries by leverage the collective power of all the developers out there that
are using tools like [goreleaser](https://goreleaser.com/) and [cargo-dist](https://github.com/axodotdev/cargo-dist)
and many others to pre-compile their software and put their binaries up on GitHub or GitLab.

## Documentation

[Full Documentation](https://dist.sh)

## Features

- Simple to install binaries on your system from multiple sources
- No reliance on a centralized repository of metadata like package managers
- Support multiple platforms and architectures
- Support private repositories (this was a feature removed from homebrew)
- Support checksum verifications (if they exist)
- Support signatures verifications (if they exist)

## Quickstart

See full documentation at [Installation](https://dist.sh/installation/)

**Note:** the installation script **DO NOT CURRENTLY** try to modify your path, you will need to do that manually.

### MacOS/Linux
```bash
curl --proto '=https' --tlsv1.2 -LsSf https://get.dist.sh | sh
```

### Windows
```powershell
iwr https://get.dist.sh/install.ps1 -useb | iex
```

### Adjust Your Path

#### MacOS/Linux

```bash
export PATH=$HOME/.distillery/bin:$PATH
```

#### Windows

```powershell
[Environment]::SetEnvironmentVariable("Path", "C:\Users\<username>\.distillery\bin;" + $env:Path, [EnvironmentVariableTarget]::User)
```

## Behaviors

- Allow for multiple versions of a binary using `tool@version` syntax
- Running installation for any version will automatically update the default symlink to that version (i.e. switching versions)
- Caching of HTTP calls where possible (GitHub primarily)
- Caching of downloads

### Running install always updates default symlink

**Note:** this might change before exiting beta.

Whenever you run install the default symlink will always be updated to whatever version you specify. This is to make
it easy to switch versions.

### Multiple Versions

Every time you run install it will by default seek out the latest version, it will not remove any other versions. All
versions are symlinked with the suffix `@version` this means you can have multiple versions installed at the same time.

It also means you can call any version any time using the `@version` syntax or if you are using something like [direnv](https://direnv.net/)
you can set aliases in your `.envrc` file for specific versions.

#### Example

```console
alias terraform="terraform@1.8.5"
```

### Examples

Install a specific version of a tool using `@version` syntax. `github` is the default scope, this implies
`github/ekristen/aws-nuke`

```console
dist install ekristen/aws-nuke@3.16.0
```

Install a tool from a specific owner and repository, in this case hashicorp. This will install the latest version.
However, because hashicorp hosts their binaries on their own domain, distillery has special handling for obtaining
the latest version from releases.hashicorp.com instead of GitHub.

```console
dist install hashicorp/terraform
```

Install a binary from GitLab.
```console
dist install gitlab/gitlab-org/gitlab-runner
```

Often times installing from GitHub or GitLab is sufficient, but if you are on a MacOS system and Homebrew
has the binary you want, you can install it using the `homebrew` scope. I would generally still recommend just
installing from GitHub or GitLab directly.

```console
dist install homebrew/opentofu
```

## Supported Sources

- GitHub
- GitLab
- Homebrew (binaries only, if anything has a dependency, it will not work at this time)
- Hashicorp (special handling for their releases, pointing to GitHub repos will automatically pass through)
- Kubernetes (special handling for their releases, pointing to GitHub repos will automatically pass through)

### Authentication

Distillery supports authentication for GitHub and GitLab. There are CLI options to pass in a token, but the preferred
method is to set the `DISTILLERY_GITHUB_TOKEN` or `DISTILLERY_GITLAB_TOKEN` environment variables using a tool like
[direnv](https://direnv.net/).

## Directory Structure

This is the default directory structure that distillery uses. Some of this can be overridden via the configuration.

- Binaries
  - Symlinks `$HOME/.distillery/bin` (this should be in your `$PATH` variable)
  - Binaries `$HOME/.distillery/opt` (this is where the raw binaries are stored and symlinked to)
    - `source/owner/repo/version/<binaries>`
      - example: `github/ekristen/aws-nuke/v2.15.0/aws-nuke`
      - example: `hashicorp/terraform/v0.14.7/terraform`
- Cache directory (downloads, http caching)
  - MacOS `$HOME/Library/Caches/distillery`
  - Linux `$HOME/.cache/distillery`
  - Windows `$HOME/AppData/Local/distillery`

### Caching

At the moment there are two discrete caches. One for HTTP requests and one for downloads. The HTTP cache is used to
store the ETag and Last-Modified headers from the server to determine if the file has changed. The download cache is
used to store the downloaded file. The download cache is not used to determine if the file has changed, that is done
by the HTTP cache.

If you need to delete your cache simply run `dist info` identify the cache directory and remove it.

**Note:** I may add a cache clear command in the future.
