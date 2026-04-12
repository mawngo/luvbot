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

Alternately, check the [Releases](https://github.com/mawngo/luvbot/releases) page for pre-built binaries.

## Usage

### Account Setup

Run the following command will open a browser for you to set up your account.

```shell
luvbot profile setup
```

Navigate to [Instagram](https://instagram.com) and login. Wait for IG to ask for notification permission, click accept
and then reject when browser ask for allowing notification. This ensures that IG won't ask for notification again, as
the bot can't handle the notification asking popup.

The profile data will be saved to the `profiles` directory. If a name is not specified using `--profile` flag, the bot
will use the default profile.

### Run the bot

```shell
luvbot ig
```

If any error happen during the run, the bot will take a screenshot of the current page and save it to the `errors`
directory before it exit.

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
- The automatic liking is not yet reliable, it may miss some stories and occasionally some posts.
