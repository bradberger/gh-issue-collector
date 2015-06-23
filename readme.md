A [Golang](//golang.org) powered API for posting issues to GitHub.
The idea is to set up a public/semi-public issue collector and post
to a issues GitHub project without requiring a GitHub account.

## Project Configuration

## GitHub API Token

You should set each project to use it's own GitHub API token.

For the hosted version of the app, we'll use GitHub OAuth2 flow and
store the app token, but for now just use
[a personal API token](https://github.com/settings/tokens)
if you're controlling the server. It only needs the `repo` scope.

### Access Control

Each project can be set up to allow access from a list of domains, or
all domains inclusively.


## Docker

The project contains a `Dockerfile` that's designed to be run with
the [official `golang` docker image](https://registry.hub.docker.com/_/golang/).

It uses the [`onbuild` variant](https://github.com/docker-library/golang/blob/9ff2ccca569f9525b023080540f1bb55f6b59d7f/1.3.1/onbuild/Dockerfile)
for even simpler configuration.

### Configuration on Ubuntu

With `14.10`, at least, you'll need to pull the `golang:1.3-onbuild` image first:

```bash
docker pull golang:1.3-onbuild
```

### Building

From the project base directory:

```bash
docker build -t gh-issue-collector .
```

### Running

From the project base directory:

```bash
docker run --publish 6060:3000 --name gh-issue-collector --rm gh-issue-collector
```
