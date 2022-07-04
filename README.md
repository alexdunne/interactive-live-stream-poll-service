# Interactive Live Stream Poll Service

## Background Information

[Architecture Diagram](https://lucid.app/lucidchart/3fb1c616-c51d-48c0-80db-ff5b60394038/edit?invitationId=inv_a9678cf9-5e5e-4685-9307-a6e41fdcc9a0)

## Requirements

These dependencies also be installed with [Homebrew](https://brew.sh/).

- Requires Go 1.18 or greater. This can be installed with brew `brew install go` or downloaded [here](https://golang.org/doc/install).
- Requires Docker and Docker Compose. This can be installed with brew `brew install docker` or downloaded [here](https://docs.docker.com/engine/install/).
- Requires AWS SAM CLI. This can be installed with brew `brew tap aws/tap` followed by `brew install aws-sam-cli`

### Install Dependencies

Install dependencies, issue the following command(s):

```bash
make install
```

## Local development

Start watching for file changes

```bash
make watch
```

Spin up the an API locally:

```bash
sam local start-api
```
