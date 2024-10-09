![Build Status](https://img.shields.io/github/actions/workflow/status/root4loot/crtsher/test.yml) ![Go version](https://img.shields.io/badge/Go-v1.21-blue.svg) [![Contribute](https://img.shields.io/badge/Contribute-Welcome-green.svg)](CONTRIBUTING.md)

# crtsher

A tool used to grab domains from certificate transparency logs (crt.sh).

## Why another crt.sh tool?

Unlike other tools that often make a single request to crt.sh, this tool is designed to handle the inherent slowness and unreliability of crt.sh, especially when dealing with large responses. It includes retry logic to detect and recover from failed requests. It offers a simple API that can also be used to run tasks asynchronously.

## Installation

Requires Go 1.20 or later.

```bash
go install github.com/root4loot/cmd/crtsher@latest
```

## Docker

```bash
git clone https://github.com/root4loot/crtsher
cd crtsher
docker run --rm -it $(docker build -q .) example.com
```

## Usage

```bash
Usage: crtsher [options] <domain | orgname> 
  -f, --file <file>           Specify input file containing targets, one per line.
  -t, --timeout <seconds>     Set the timeout for each request (default: 90).
  -c, --concurrency <number>  Set the number of concurrent requests (default: 3).
      --debug                 Enable debug mode.
      --version               Display the version information.
      --help                  Display this help message.

Search Query Identity:
  - Domain Name
  - Organization Name

Examples:
  crtsher example.com
  crtsher "Hackerone Inc"
  crtsher --file domains.txt
```

## Example Running

```bash
$ crtsher example.com
[crtsher] (INF) Querying example.com
[crtsher] (RES) www.example.org
[crtsher] (RES) hosted.jivesoftware.com
[crtsher] (RES) uat3.hosted.jivesoftware.com
[crtsher] (RES) www.example.com
[crtsher] (RES) example.com
```

## As a Library

See the `examples` folder for usage examples.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)
