# Configuration

## Zero Configuration Philosophy

**distillery** is designed to work out of the box with no configuration for most users. **distillery** has sensible
defaults.

## Location

The default location for the configuration file is operating system dependent. YAML and TOML are supported.

- On macOS, it is located at `~/.config/distillery.yaml`
- On Linux, it is located at `~/.config/distillery.yaml`.
- On Windows, it is located at `%APPDATA%\distillery.yaml`

The configuration file is optional. If it is not found, the default configuration is used.

!!! note - "Pro Tip"
    You can change the default location of your configuration file by setting the `DISTILLERY_CONFIG` environment variable.

## Default Configuration

=== "YAML"

    ```yaml
    default_provider: github
    ```

=== "TOML"

    ```toml
    default_provider = "github"
    ```

## Aliases

Aliases are useful shortcuts for repositories. See [Aliases](config/aliases.md) for more information.

## Settings

- `checksum-missing` (string): This is the behavior when a checksum is missing. The default is `warn`, valid values are `warn`, `error`, and `ignore`.
- `checksum-unknown` (string): This is the behavior when a checksum method is unknown. The default is `warn`, valid values are `warn`, `error`, and `ignore`.
- `signature-missing` (string): This is the behavior when a signature is missing. The default is `warn`, valid values are `warn`, `error`, and `ignore`.

=== "YAML"

    ```yaml
    settings:
      checksum-missing: warn
      checksum-unknown: warn
      signature-missing: warn
    ```

=== "TOML"

    ```toml
    [settings]
    checksum-missing = "warn
    checksum-unknown = "warn"
    signature-missing = "warn"
    ```