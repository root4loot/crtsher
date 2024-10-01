package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

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

const usage = `
Usage: crtsher [options] <domain | orgname> 
  -f, --file <file>           Specify input file containing targets, one per line.
  -t, --timeout <seconds>     Set the timeout for each request (default: 90).
  -c, --concurrency <number>  Set the number of concurrent requests (default: 3).
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
	inputList, opts, err := parseCLI()
	runner := crtsher.NewRunnerWithOptions(opts)

	if err != nil {
		if err.Error() == "version" {
			fmt.Println("version:", Version)
			os.Exit(0)
		}

		log.Error(err)
		os.Exit(0)
	}

	if hasStdin() {
		log.Debug("Reading from stdin")

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inputList = append(inputList, strings.TrimSpace(scanner.Text()))
		}
	}

	if inputList != nil {
		processResults(runner, inputList...)
	}

}

func processResults(runner *crtsher.Runner, target ...string) {
	go runner.RunMultipleAsync(target)

	printed := make(map[string]bool)
	for res := range runner.Results {
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

func parseCLI() ([]string, *crtsher.Options, error) {
	var version, help bool
	var inputFilePath string
	var inputList []string

	opts := *crtsher.DefaultOptions()

	flag.StringVar(&inputFilePath, "f", "", "")
	flag.StringVar(&inputFilePath, "file", "", "")
	flag.IntVar(&opts.Timeout, "timeout", opts.Timeout, "")
	flag.IntVar(&opts.Timeout, "t", opts.Timeout, "")
	flag.IntVar(&opts.Concurrency, "concurrency", opts.Concurrency, "")
	flag.IntVar(&opts.Concurrency, "c", opts.Concurrency, "")
	flag.BoolVar(&version, "version", false, "")
	flag.BoolVar(&help, "help", false, "")
	flag.Usage = func() {
		fmt.Fprint(os.Stdout, usage)
	}
	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		inputList = args
	}

	if inputFilePath != "" {
		lines, err := fileutil.ReadFile(inputFilePath)
		if err != nil {
			return nil, nil, err
		}

		inputList = append(inputList, lines...)
	}

	if help || version || (len(flag.Args()) == 0 && len(inputList) == 0 && !hasStdin()) {
		if help {
			fmt.Fprint(os.Stdout, usage)
			return nil, nil, nil
		} else if version {
			log.Info("Version:", Version)
		} else {
			return nil, nil, fmt.Errorf("No input provided. See -h for usage")
		}
	}

	if log.IsOutputPiped() {
		log.Notify(log.PipedOutputNotification)
	}

	if hasStdin() {
		inputList = append(inputList, processStdin()...)
	}

	if inputFilePath != "" {
		lines, err := fileutil.ReadFile(inputFilePath)
		if err != nil {
			return nil, nil, err
		}

		inputList = append(inputList, lines...)
	}

	return inputList, &opts, nil
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

func processStdin() []string {
	var targets []string
	if hasStdin() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) > 0 {
				targets = append(targets, strings.Fields(line)...)
			}
		}
	}
	return targets
}
