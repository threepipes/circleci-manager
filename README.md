[![threepipes](https://circleci.com/gh/threepipes/circleci-env.svg?style=svg)](https://github.com/threepipes/circleci-env)

# ccienv

A management tool for CircleCI Project's environment variables.  
(Currently, only available on GitHub)

This repository will be supported until the official circleci-cli will support the project's variables management: [issue](https://github.com/CircleCI-Public/circleci-cli/issues/652)

## Installation

```
$ go install github.com/threepipes/circleci-env/cmd/ccienv@latest
```

### Uninstallation

```
$ rm $(which ccienv)
```

## Requirements

- golang
- git
    - Only if load repository name from .git

## Setup

```
$ ccienv config init
```

Set these variables.

- CircleCI API Token
    - A personal API token of CircleCI
- GitHub organization
    - GitHub organization name or GitHub username of your repository

Then, `$XDG_CONFIG_HOME/ccienv/config.yml` will be created.

## Run

```
$ ccienv -r <your_repo_name> <cmd> [<args>]
```

If `-r <your_repo_name>` is omitted, the origin URL of the current directory's git project is used to specify the target repository.  
It is the same as the result of `git config --get remote.origin.url`.
Then, you can use ccienv like this.
```
# You have to be in a directory of a target repository
$ ccienv ls
```

### Example

```
# list variables
$ ccienv -r circleci-env ls

# Add a variable
$ ccienv add TEST_ENV somevalue

# Delete variables interactive
$ ccienv rm -i
```

## Help

You can find more information by this command.
```
$ ccienv -h
```
