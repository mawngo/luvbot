# Luv Bot

A bot that automatically likes Instagram posts and stories.

- To make your friends happy.
- To spread love to the world.

**Using this bot may get your account banned.**

## Installation

Require go 1.26+

```shell
go install github.com/mawngo/luvbot@latest
```

## Usage

### Account Setup

Run the following command will open a browser for you to set up your account.

```shell
luvbot profile setup
```

### Run the bot

```shell
luvbot ig
```

### Options

```
> luvbot -h  
Automatically liking Instagram posts and stories

Usage:
  luvbot [command]

Available Commands:
  profile     Profile account management
  ig          Automatically liking Instagram posts and stories
  help        Help about any command
  completion  Generate the autocompletion script for the specified shell

Flags:
      --debug   enable debug mode
  -h, --help    help for luvbot

Use "luvbot [command] --help" for more information about a command.
```

## Caveats

- Using the bot could get your account banned.
- Sometimes the bot may fail to detect stories/posts and freeze.
- The automatic posts liking may not work as expected on videos (yet).
- The automatic stories liking is not reliable (yet), it could miss some stories.