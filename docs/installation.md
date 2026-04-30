# Installation

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

## Manual Installation

## Unix-like Systems

macOS, Linux, FreeBSD and other Unix-like systems can use the following steps to install distillery.

1. Set your path `export PATH=$HOME/.distillery/bin:$PATH`
2. Download the latest release from the [releases page](https://github.com/ekristen/distillery/releases)
3. Extract and Run `./dist install ekristen/distillery`
4. Delete `./dist` and the .tar.gz, now use `dist` normally
5. Run `dist install owner/repo` to install a binary from GitHub Repository

### Windows

1. [Set Your Path](#set-your-path)
2. Download the latest release from the [releases page](https://github.com/ekristen/distillery/releases)
3. Extract and Run `.\dist.exe install ekristen/distillery`
4. Delete `.\dist.exe` and the .zip, now use `dist` normally
5. Run `dist install owner/repo` to install a binary from GitHub Repository

#### Set Your Path

##### For Current Session

```powershell
$env:Path = "C:\Users\<username>\.distillery\bin;" + $env:Path
```

##### For Current User

```powershell
[Environment]::SetEnvironmentVariable("Path", "C:\Users\<username>\.distillery\bin;" + $env:Path, [EnvironmentVariableTarget]::User)
```

## Uninstall

By default, everything with distillery is installed in the `~/.distillery` directory so cleanup is easy.

1. Run `dist info`
2. Remove the directories listed under the cleanup section
