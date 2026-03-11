## Quickstart

=== "macOS/Linux"
    ```bash
    curl --proto '=https' --tlsv1.2 -LsSf https://get.dist.sh | sh
    ```

=== "Windows"
    ```powershell
    iwr https://get.dist.sh/install.ps1 -useb | iex
    ```

**Note:** Yes, I know, you really shouldn't download and run scripts from the internet, but at least it's not using `sudo`!

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

## Features

- Simple to install binaries on your system from multiple sources
- No reliance on a centralized repository of metadata like package managers
- Support multiple platforms and architectures
- Support private repositories (this was a feature removed from homebrew)
- Support checksum verifications (if they exist)
- Support signatures verifications (if they exist)
- [Aliases](config/aliases.md) for easy access to binaries

## Examples

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

Often times installing from GitHub or GitLab is sufficient, but if you are on a macOS system and Homebrew
has the binary you want, you can install it using the `homebrew` scope. I would generally still recommend just
installing from GitHub or GitLab directly.

```console
dist install homebrew/opentofu
```

## Behaviors

- Allow for multiple versions of a binary using `tool@version` syntax
- Running installation for any version will automatically update the default symlink to that version (i.e. switching versions)
- Caching of HTTP calls where possible (GitHub primarily)
- Caching of downloads

### Running install always updates default symlink

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
