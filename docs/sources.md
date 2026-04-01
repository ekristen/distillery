# Supported Sources

- GitHub
- GitLab
- Forgejo / Codeberg (Codeberg works out of the box; any Forgejo instance can be configured)
- Homebrew (binaries only, if anything has a dependency, it will not work at this time)
- Hashicorp (special handling for their releases, pointing to GitHub repos will automatically pass through)
- Kubernetes (special handling for their releases, pointing to GitHub repos will automatically pass through)

## Authentication

Distillery supports authentication for GitHub, GitLab, and Forgejo/Codeberg. There are CLI options to pass
in a token, but the preferred method is to set the appropriate environment variable using a tool like
[direnv](https://direnv.net/).

| Source | Environment variable | CLI flag |
|---|---|---|
| GitHub | `DISTILLERY_GITHUB_TOKEN` | `--github-token` |
| GitLab | `DISTILLERY_GITLAB_TOKEN` | `--gitlab-token` |
| Forgejo / Codeberg | `DISTILLERY_FORGEJO_TOKEN` | `--forgejo-token` |

This allows you to bypass any API rate limits that might be in place for unauthenticated requests, but more
importantly it allows you to install from private repositories that you have access to!

The same `--forgejo-token` / `DISTILLERY_FORGEJO_TOKEN` is used for authentication across all
configured Forgejo providers.

## Forgejo / Codeberg

Codeberg is supported via the `codeberg` prefix; no configuration is required.

Any self-hosted Forgejo (or Gitea-compatible) instance can be added as a named provider in your
configuration file. The provider name you choose becomes the routing prefix used on the command line.

=== "YAML"

    ```yaml
    providers:
      myforgejo:
        provider: forgejo
        base_url: https://git.example.com/api/v1
    ```

=== "TOML"

    ```toml
    [providers.myforgejo]
    provider = "forgejo"
    base_url = "https://git.example.com/api/v1"
    ```

Once configured, use the provider name as the prefix:

```console
dist install myforgejo/owner/repo
dist install myforgejo/owner/repo@2.0.0
```
