![Go version](https://img.shields.io/badge/Go-v1.19-blue.svg) [![Contribute](https://img.shields.io/badge/Contribute-Welcome-green.svg)](CONTRIBUTING.md)

# ctlog
A package used to obtain domains from transparancy logs, either by domain or organization name.

```
go get github.com/root4loot/ctlog@latest
```

See [Examples](https://github.com/root4loot/ctlog/tree/master/examples)

```go
package main

import (
	"fmt"

	"github.com/root4loot/ctlog"
)

func main() {
	// run ctlog against targets
	results := ctlog.Multiple([]string{"example.com", "Hackerone Inc"})
	for _, result := range results {
		for _, res := range result {
			if res.Domain() != "" {
				fmt.Println(res.Domain())
			}
		}
	}
}
```

### Output

```
hackerone.com
enorekcah.com
errors.hackerone.net
gitaly.code-pdx1.inverselink.com
www.testserver.inverselink.com
www.enorekcah.com
www.hackerone.com
events.hackerone.com
go.inverselink.com
support-app.inverselink.com
staging.inverselink.com
testserver.inverselink.com
attjira.inverselink.com
signatures.hacker.one
looker.inverselink.com
links.hackerone.com
support.hackerone.com
phabricator.inverselink.com
ci.inverselink.com
info.hackerone.com
hackerone-user-content.com
hackerone-ext-content.com
ci-production.inverselink.com
storybook.inverselink.com
go.hacker.one
sentry.inverselink.com
ma.hacker.one
payments-production.inverselink.com
hacker.one
ui-docs.inverselink.com
proteus.inverselink.com
info.hacker.one
logstash.inverselink.com
kibana.inverselink.com
withinsecurity.com
bd1.inverselink.com
bd2.inverselink.com
bd3.inverselink.com
www.example.org
hosted.jivesoftware.com
uat3.hosted.jivesoftware.com
www.example.com
example.com
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)
