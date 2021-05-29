[![Go](https://github.com/rainu/mqtt-shell/actions/workflows/build.yml/badge.svg)](https://github.com/rainu/mqtt-shell/actions/workflows/build.yml)

# mqtt-shell

A shell like command line interface for mqtt written in go. With it, you can easily subscribe and publish mqtt topics. 
It is also possible to pass through the incoming messages to external applications. Such like piping in shells!

![](./doc/example.gif)

Features:
* Subscribe (multiple) mqtt topics
* Publish messages to mqtt topic
* Pipe the incoming messages to external applications
* Command history (such like other shells)
* Configuration support via yaml-files
    * so you can use handle multiple environments easily
* Macro support

# Get the Binary
You can build it on your own (you will need [golang](https://golang.org/) installed):
```bash
go build -a -installsuffix cgo ./cmd/mqtt-shell/
```

Or you can download the release binaries: [here](https://github.com/rainu/mqtt-shell/releases/latest)

# Usage 

To see all command options, simple type
```bash
./mqtt-shell -h

Usage of ./mqtt-shell:
  -b string
    	The broker URI. ex: tcp://127.0.0.1:1883
  -c string
    	The ClientID (default "mqtt-shell")
  -ca string
    	MQTT ca file path (if tls is used)
  -cmd value
    	The command(s) which should be executed at the beginning
  -cs
    	Indicating that no messages saved by the broker for this client should be delivered (default true)
  -e string
    	The environment which should be used
  -ed string
    	The environment directory (default "~/.mqtt-shell")
  -hf string
    	The history file path (default "~/.mqtt-shell/.history")
  -ni
    	Should this shell be non interactive. Only useful in combination with 'cmd' option
  -p string
    	The password
  -pq int
    	The default Quality of Service for publishing 0,1,2 (default 1)
  -sp string
    	The prompt of the shell (default "\\033[36m»\\033[0m ")
  -sq int
    	The default Quality of Service for subscription 0,1,2
  -u string
    	The username
```

## Environment configurations

You can create yaml files where you can configure predefined configuration. This can be helpful for different mqtt environments.
This files must be stored in the environment directory (by default ~/.mqtt-shell/).

For example:
```yaml
broker: tls://127.0.0.1:8883
ca: /tmp/my.ca
subscribe-qos: 1
publish-qos: 2
username: user
password:  secret
client-id: my-mqtt-shell
clean-session: true
commands: 
  - sub #
non-interactive: false
history-file: /home/user/.mqtt-shell/history
prompt: "\033[36mmsh>\033[0m "
macros:
  my-macro:
    description: Awesome description of my macro
    arguments:
      - message
    commands:
      - pub test $1
```
