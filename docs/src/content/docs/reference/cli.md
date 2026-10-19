---
title: Command line
description: Commands and flags of the swim-timer binary.
sidebar:
  order: 2
---

Without a command, the binary starts Swim-Timer. Commands run a single task and exit with code 0 on success
and 1 on an error.

```sh
swim-timer                    # start Swim-Timer
swim-timer --help             # list all commands and flags
swim-timer <command> --help   # show the flags and examples of one command
```

Every [setting](./settings.md) is also a flag of every command, such as `--server-port`.
`--config` sets the path of the config file.

| Command                            | Description                                                                         |
| ---------------------------------- | ----------------------------------------------------------------------------------- |
| `config init [file]`               | Write a config file with every setting at its default, each explained by a comment. |
| `config path`                      | Show the config file in use and the locations searched for one, in order.           |
| `config show`                      | Print the effective configuration with secrets masked.                              |
| `config validate [file]`           | Check the configuration and report every problem.                                   |
| `healthcheck`                      | Check that Swim-Timer answers on this computer.                                     |
| `recording csv <recording> [file]` | Convert a recording of the timing data into a CSV table.                            |
| `update check`                     | Check whether a newer release exists.                                               |
| `update apply`                     | Install the latest release, see [Updates](../installation/updates.md).              |
| `version`                          | Print the version.                                                                  |
| `gen man [directory]`              | Generate man pages for every command.                                               |
| `completion <shell>`               | Print a shell completion script for bash, zsh, fish, or powershell.                 |

## Configuration

| Command           | Options                                                                                              |
| ----------------- | ---------------------------------------------------------------------------------------------------- |
| `config init`     | Writes to the given file or the standard output. `--force` replaces an existing file.                |
| `config show`     | `--output json` or `--output table` change the format, `--origin` shows the source of every value.   |
| `config validate` | Checks the configuration Swim-Timer would start with, or only the given file on top of the defaults. |

## Health check

`healthcheck` exits with 0 when Swim-Timer answers on this computer. The container image uses it as its health check.

## Recordings

`recording csv` converts a [recording](./recordings.md) into a [table for analysis](./recordings.md#table-for-analysis):

```sh
swim-timer recording csv <recording> <file>
```

## Shell completion and man pages

`completion` prints a completion script for the commands, the flags, and settings with fixed values.
`gen man` writes a man page per command. The Debian package installs both.
