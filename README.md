# Tinystat 

[![CircleCI](https://circleci.com/gh/sdwolfe32/tinystat.svg?style=svg)](https://circleci.com/gh/sdwolfe32/tinystat)
[![GoDoc](https://godoc.org/github.com/sdwolfe32/tinystat/client?status.svg)](https://godoc.org/github.com/sdwolfe32/tinystat/client)

Tinystat is a free and open source (very) minimalist stats API. The need for a basic statistics system came when I was doing some recent (fairly large) COUNT queries in MySQL. These queries were done over 6M+ records and thus resulted in massive response times. Rather than creating a count table for every app that I build, I decided to build a counting app! And so, Tinystat was born.

The API itself is available in two forms, the public API (more info: https://tinystat.io), and a public Docker image on DockerHub (see: https://hub.docker.com/r/sdwolfe32/tinystat/). You can choose to either use the public one or host it yourself.

## Using the API (public or self-hosted)

All routes are outlined in main.go (see: https://github.com/sdwolfe32/tinystat/blob/master/main.go#L46-L50).

## Public API limitations

- All POST requests are rate limited to 1RPS to help prevent flooding inserts. (No GET requests are rate limited)
- Each IP is limited to 5 apps by default, again to help prevent flooding the apps table with inserts.

## Using the client library

```go
package main

import (
    "log"
    "time"

	tinystat "github.com/sdwolfe32/tinystat/client"
)

func main() {
  // FIRST - Make sure the following environment variables are set
  // TINYSTAT_APP_ID - The appID created on the Tinystat API
  // TINYSTAT_TOKEN - the token provided when the app was generated

  // Now lets report a few actions called "example-action"
  tinystat.CreateAction("example-action")
  tinystat.CreateAction("example-action")
  tinystat.CreateAction("example-action")
  
  // Be sure to wait at least 10 seconds to let the periodic reporting script report
  time.Sleep(11 * time.Second)

  // Now lets get a summary of the action!
  log.Println(tinystat.GetActionSummary("example-action"))
  
  // You can also get a count in a specified duration
  // NOTE: Must follow the spec - https://golang.org/pkg/time/#ParseDuration
  log.Println(tinystat.GetActionCount("example-action", "5h"))
}
```

## Running with Docker

```
docker run -p 8080:8080 -e MYSQL_URL={{YourMySQLURL}} sdwolfe32/tinystat
```

The BSD 3-clause License
========================

Copyright (c) 2018, Steven Wolfe. All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

 - Redistributions of source code must retain the above copyright notice,
   this list of conditions and the following disclaimer.

 - Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

 - Neither the name of Tinystat nor the names of its contributors may
   be used to endorse or promote products derived from this software without
   specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.