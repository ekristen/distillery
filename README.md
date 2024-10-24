# Go Project Template

This is an opinionated go project template to use as a starting point for new projects.

## Features

- Builds with [GoReleaser](https://goreleaser.com)
  - Automated with GitHub Actions
  - Signed with Cosign (providing you generate a private key)
- Linting with [golangci-lint](https://golangci-lint.run/)
  - Automated with GitHub Actions
- Builds with Docker
  - While designed to use goreleaser, you can still just run `docker build`
- Apple Notary Signing Support
- Opinionated Layout
  - Never use `internal/` folder
  - Everything is under `pkg/` folder
- Commits must meet [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
  - Automated with GitHub Actions ([commit-lint](https://github.com/conventional-changelog/commitlint/#what-is-commitlint))
- Automatic Dependency Management with [Renovate](https://github.com/renovatebot/renovate)
- Automatic [Semantic Releases](https://semantic-release.gitbook.io/)
- Documentation with Material for MkDocs
- API Server Example
  - Uses Gorilla Mux (yes it's been archived, still the best option)
- Stubbed out Go Tests (**note:** they are not comprehensive)

### Opinionated Decisions

<<<<<<< Updated upstream
- Uses `init` functions for registering commands globally.
  - This allows for multiple `main` package files to be written and include different commands.
  - Allows the command code to remain isolated from each other and a simple import to include the command.
=======
- [ ] detect already installed versions
- [ ] implement signature verification
- [ ] implement uninstall
>>>>>>> Stashed changes

### Multi-Platform Builds

This project is designed to build for multiple platforms, including macOS, Linux, and Windows. It also supports
multiple architectures including amd64 and arm64. 

The goreleaser configuration is set up to build for all platforms and architectures by default. It even supports pushing
multi-architecture docker manifests by default. Some knowledge about GoReleaser's configuration is required should you
want to remove these capabilities.

### Apple Notary Signing

This makes use of a tool called [quill](https://github.com/anchore/quill). To make use of this feature you will need
to have an Apple Developer account and be able to create an Developer ID certificate.

The workflow is designed to pull the necessary secrets from 1Password. This is done to keep the secrets out of the
GitHub Actions logs. The secrets are pulled from 1Password if the event triggering the workflow is a tag **AND** the
actor is the owner of the repository. This is to prevent forks from being able to pull the secrets and is an extra
guard to help prevent theft.

GoReleaser is configured to always sign and notarize for macOS. However, it will not notarize if the build is a snapshot.

If configured properly, the binaries located within the archives produced by GoReleaser will be signed and notarized
by the Apple Notary Service and will automatically run on any macOS system without having to approve it under System
Preferences.

If you do not wish to use 1Password simply export the same environment variables using secrets to populate them. The 
`QUILL_SIGN_P12` and `QUILL_NOTARY_KEY` need to be base64 encoded or paths to the actual files.

## Building

The following will build binaries in snapshot order.

```console
goreleaser --clean --snapshot --skip sign
```

**Note:** we are skipping signing because this project uses cosign's keyless signing with GitHub Actions OIDC provider.

You can opt to generate a cosign keypair locally and set the following environment variables, and then you can run
`goreleaser --clean --snapshot` without the `--skip sign` flag to get signed artifacts.

Environment Variables:
- 
- COSIGN_PASSWORD
- COSIGN_KEY (path to the key file) (recommend cosign.key, it is git ignored already)

```console
cosign generate-key-pair
```

## Configure

1. Rename Repository
2. Generate Cosign Keys (optional if you want to run with signing locally, see above)
3. Update `.goreleaser.yml`, search/replace go-project-template with new project name, adjust GitHub owner
4. Update `main.go`,
5. Update `go.mod`, rename go project (using IDE is best so renames happen across all files)

### Docker

The Dockerfile is set up to build the project and then copy the artifacts from the build into the final image. It is
also configured to allow you to just run `docker build` directly if you do not want to use GoReleaser. 

To make things easier and faster, the Dockerfile has a default build argument set to `go-project-template`. GoReleaser
will pass the new project name down (if you update the `.goreleaser.yml` file) and the Dockerfile will use that instead.

However, it would be better longer term to update this argument in the file or remove it all together.

### Signing

Signing happens via cosign's keyless features using the GitHub Actions OIDC provider.

### Releases

In order for Semantic Releases and GoReleaser to work properly you have to create a PAT to run Semantic Release
so it's actions against the repository can trigger other workflows. Unfortunately there is no way to trigger
a workflow from a workflow if both are run by the automatically generated GitHub Actions secret.

1. Create PAT that has content `write` permissions to the repository
2. Create GitHub Action Secret
   - `SEMANTIC_GITHUB_TOKEN` -> populated with PAT from step 1
3. Done

## Documentation

The project is built to have the documentation right alongside the code in the `docs/` directory leveraging Mkdocs Material.

In the root of the project exists mkdocs.yml which drives the configuration for the documentation.

This README.md is currently copied to `docs/index.md` and the documentation is automatically published to the GitHub
pages location for this repository using a GitHub Action workflow. It does not use the `gh-pages` branch.

### Running Locally

```console
make docs-serve
```

OR (if you have docker)

```console
docker run --rm -it -p 8000:8000 -v ${PWD}:/docs squidfunk/mkdocs-material
```
