# Distfile

A Distfile is similar to a Dockerfile or a Brewfile, it's a simple list of instructions to install software on a system.

- All instructions are executed in order.
- Instructions are case-insensitive.
- Configuration files are honored during installation.

## Building

The command to build a Distfile based on your current installed software is `proof` but is as the alias of `export`.

- **alias:** - `export`

```bash
dist proof
```

## Running

!!! note
    If you do not pass in a distfile, it will look for a `Distfile` in the current directory by default, followed
    by `Distfile` in your home directory.

```console
dist run <distfile>
```

### Parallel Installs

!!! warning
    Experimental Feature - it might not work correctly and the CLI output will be all mixed together. Expect the CLI 
    output to change in future versions.

Currently, it takes an integer value for the number of parallel installations to run. 

```console
dist run --parallel <number> [distfile]
```

## Instructions

These are the current supported instructions.

- `install` - Install a package from a distribution source.
- `file` - Include another distfile.

### Install Instruction

All validate arguments to the `install` command line instruction are valid as arguments to the `install` instruction.

### Example

```distfile
install github/ekristen/aws-nuke@v3.39.0

file ~/.config/distfiles/extra
```
