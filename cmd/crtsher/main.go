package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/root4loot/crtsher"
	"github.com/root4loot/goutils/domainutil"
	"github.com/root4loot/goutils/fileutil"
	"github.com/root4loot/goutils/log"
)

const (
	AppName = "crtsher"
	Version = "0.1.0"
)

func init() {
	log.Init(AppName)
}

type CLI struct {
	Target      string
	Concurrency int
	Infile      string
	Timeout     int
	Debug       bool
}

func NewCLI() *CLI {
	return &CLI{
		Concurrency: 3,
		Timeout:     90,
	}
}

// TODO: implement debug flag
const usage = `
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
`

func main() {
	cli := NewCLI()
	cli.parseFlags()
	flag.Parse()

	if len(flag.Args()) == 0 && !hasStdin() && cli.Infile == "" {
		log.Error("No targets specified. Please provide a domain or organization name.")
		fmt.Print(usage)
		os.Exit(0)
	}

	targetChannel := make(chan string)

	go cli.processTargets(targetChannel)
	cli.processTarget(targetChannel) // Directly call without a done channel
}

func (cli *CLI) processTargets(targetChannel chan<- string) {
	args := flag.Args()

	if len(args) > 0 {
		log.Debug("Processing command line arguments")
		for _, target := range args {
			targetChannel <- target
		}
	}

	if hasStdin() {
		log.Debug("Reading targets from standard input")
		cli.processInput(os.Stdin, targetChannel)
	}

	if cli.Infile != "" {
		log.Debugf("Reading targets from file: %s", cli.Infile)
		cli.processFileTargets(cli.Infile, targetChannel)
	}

	close(targetChannel)
}

func (cli *CLI) processInput(input io.Reader, targetChannel chan<- string) {
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		for _, target := range strings.Fields(scanner.Text()) {
			targetChannel <- target
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("Error reading input:", err)
		os.Exit(1)
	}
}

func (cli *CLI) processTarget(targetChannel <-chan string) {
	sem := make(chan struct{}, cli.Concurrency)
	var wg sync.WaitGroup

	for target := range targetChannel {
		log.Debug("Processing CLI target:", target)
		sem <- struct{}{}
		wg.Add(1)
		go func(t string) {
			defer func() { <-sem }()
			defer wg.Done()
			results := cli.worker(t)
			cli.processResults(results)
		}(target)
	}

	wg.Wait()
}

func (cli *CLI) processFileTargets(infile string, targetChannel chan<- string) {
	fileTargets, err := fileutil.ReadFile(infile)
	if err != nil {
		log.Error("Error reading file:", err)
		close(targetChannel)
		os.Exit(1)
	}
	for _, target := range fileTargets {
		targetChannel <- target
	}
}

func (cli *CLI) worker(target string) []crtsher.Result {
	crtsherRunner := crtsher.NewRunner()
	crtsherRunner.Options.Concurrency = cli.Concurrency
	crtsherRunner.Options.Timeout = cli.Timeout
	log.Debugf("Starting query for target: %s", target)
	results := crtsherRunner.Query(target)
	log.Debugf("Finished query for target: %s", target)
	return results
}

func (cli *CLI) processResults(results []crtsher.Result) {
	printed := make(map[string]bool)
	for _, res := range results {
		res.CommonName = strings.ToLower(res.CommonName)
		res.CommonName = strings.TrimPrefix(res.CommonName, "*.")
		res.IssuerName = strings.ToLower(res.IssuerName)
		res.IssuerName = strings.TrimPrefix(res.IssuerName, "*.")

		if !printed[res.CommonName] {
			if domainutil.IsDomainName(res.CommonName) {
				log.Result(res.CommonName)
				printed[res.CommonName] = true
			}

			if domainutil.IsDomainName(res.IssuerName) {
				log.Result(res.IssuerName)
				printed[res.IssuerName] = true
			}
		}
	}
}

func (cli *CLI) parseFlags() {
	var help, ver, debug bool

	opts := *crtsher.DefaultOptions()

	setStringFlag := func(name, shorthand, value, usage string) {
		flag.StringVar(&cli.Infile, name, value, usage)
		flag.StringVar(&cli.Infile, shorthand, value, usage)
	}

	setIntFlag := func(name, shorthand string, value int, usage string) {
		flag.IntVar(&cli.Timeout, name, value, usage)
		flag.IntVar(&cli.Timeout, shorthand, value, usage)
	}

	setStringFlag("file", "f", "", "")
	setIntFlag("timeout", "t", opts.Timeout, "")
	setIntFlag("concurrency", "c", opts.Concurrency, "")

	flag.BoolVar(&debug, "debug", false, "")
	flag.BoolVar(&ver, "version", false, "")
	flag.BoolVar(&help, "help", false, "")

	flag.Usage = func() {
		fmt.Fprint(os.Stdout, usage)
	}
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	if help {
		fmt.Print(usage)
		os.Exit(0)
	}

	if ver {
		fmt.Println(AppName, Version)
		os.Exit(0)
	}
}

func hasStdin() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	mode := stat.Mode()

	isPipedFromChrDev := (mode & os.ModeCharDevice) == 0
	isPipedFromFIFO := (mode & os.ModeNamedPipe) != 0

	return isPipedFromChrDev || isPipedFromFIFO
}
