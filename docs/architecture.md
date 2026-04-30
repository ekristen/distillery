# Architecture

## Directory Structure

This is the default directory structure that distillery uses. Some of this can be overridden via the configuration.

- Binaries
    - Symlinks `$HOME/.distillery/bin` (this should be in your `$PATH` variable)
    - Binaries `$HOME/.distillery/opt` (this is where the raw binaries are stored and symlinked to)
        - `source/owner/repo/version/<binaries>`
            - example: `github/ekristen/aws-nuke/v2.15.0/aws-nuke`
            - example: `hashicorp/terraform/v0.14.7/terraform`
- Cache directory (downloads, http caching)
    - macOS `$HOME/Library/Caches/distillery`
    - Linux `$HOME/.cache/distillery`
    - Windows `$HOME/AppData/Local/distillery`

### Caching

At the moment there are two discrete caches. One for HTTP requests and one for downloads. The HTTP cache is used to
store the ETag and Last-Modified headers from the server to determine if the file has changed. The download cache is
used to store the downloaded file. The download cache is not used to determine if the file has changed, that is done
by the HTTP cache.

If you need to delete your cache simply run `dist info` identify the cache directory and remove it.

**Note:** I may add a cache clear command in the future.
