![Go version](https://img.shields.io/badge/Go-v1.21-blue.svg) [![Contribute](https://img.shields.io/badge/Contribute-Welcome-green.svg)](CONTRIBUTING.md)

# ctlog

A tool used to grab domains from certificate transparency logs (crt.sh).

## Why another crt.sh tool?

Unlike other tools that often make a single request to crt.sh, this tool is designed to handle the inherent slowness and unreliability of crt.sh, especially when dealing with large responses. It includes retry logic to detect and recover from failed requests. It offers a simple API that can also be used to run tasks asynchronously.

## Installation

```bash
go get github.com/root4loot/ctlog@latest
```

## Docker

```bash
git clone https://github.com/root4loot/ctlog
cd ctlog
docker run --rm -it $(docker build -q .) example.com
```

## Usage

```bash
Usage: ctlog [options] <domain | orgname>
  -f, --file <file>           Specify input file containing targets, one per line.
  -t, --timeout <seconds>     Set the timeout for each request (default: 90).
  -c, --concurrency <number>  Set the number of concurrent requests (default: 3).
      --version               Display the version information.
      --help                  Display this help message.

Search Query Identity:
  - Domain Name
  - Organization Name

Examples:
  ctlog example.com
  ctlog "Hackerone Inc"
  ctlog --file domains.txt
```

## Example Running

```bash
$ ctlog example.com
[ctlog] (INF) Querying example.com
[ctlog] (RES) www.example.org
[ctlog] (RES) hosted.jivesoftware.com
[ctlog] (RES) uat3.hosted.jivesoftware.com
[ctlog] (RES) www.example.com
[ctlog] (RES) example.com
```

## As a Library

See the `examples` folder for usage examples.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)
