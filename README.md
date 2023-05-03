[![Go](https://github.com/rainu/mqtt-shell/actions/workflows/build.yml/badge.svg)](https://github.com/rainu/mqtt-shell/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/rainu/mqtt-shell/branch/main/graph/badge.svg)](https://codecov.io/gh/rainu/mqtt-shell)
[![Go Report Card](https://goreportcard.com/badge/github.com/rainu/mqtt-shell)](https://goreportcard.com/report/github.com/rainu/mqtt-shell)
[![Go Reference](https://pkg.go.dev/badge/github.com/rainu/mqtt-shell.svg)](https://pkg.go.dev/github.com/rainu/mqtt-shell)
# mqtt-shell

A shell like command line interface for MQTT written in go. With it, you can easily subscribe and publish MQTT topics. 
It is also possible to pass through the incoming messages to external applications. Such like piping in shells!

![](./doc/example.gif)

Features:
* Colored output
* Subscribe (multiple) MQTT topics
* Publish messages to MQTT topic
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

Or for Arch-Linux you can install the AUR-Package [mqtt-shell](https://aur.archlinux.org/packages/mqtt-shell/)
```bash
yay -S mqtt-shell
```

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
  -cb value
        This color(s) will not be used
  -cmd value
        The command(s) which should be executed at the beginning
  -cs
        Indicating that no messages saved by the broker for this client should be delivered (default true)
  -e string
        The environment which should be used
  -ed string
        The environment directory (default "~/.config/mqtt-shell")
  -hf string
        The history file path (default "~/.config/mqtt-shell/.history")
  -hh
        Show detailed help text
  -m value
        The macro file(s) which should be loaded (default [~/.config/mqtt-shell/.macros.yml])
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
  -v    Show the version
```

## Setting files

All options can be written in separate environment files (one per environment) or for global settings in the `.global.yml` file.
These files must be stores inside the shell-config directory (`~/.config/mqtt-shell`).

## Environment configurations

You can create yaml files where you can configure predefined configuration. This can be helpful for different MQTT environments.
This files must be stored in the environment directory (by default ~/.config/mqtt-shell/).

For example:
```yaml
# example.yml

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
history-file: ~/.config/mqtt-shell/history
prompt: "\033[36mmsh>\033[0m "
macros:
  my-macro:
    description: Awesome description of my macro
    arguments:
      - message
    commands:
      - pub test $1
color-blacklist:
  - "38;5;237"
```

```bash
$ ./mqtt-shell -e example
```

# multiline publishing

If you want to publish a multiline message to topic:
```bash
pub test/topic <<EOF
This is
a multiline
message
EOF
```

This is also useful if you don't want to handle argument escaping:
```bash
pub test/topic <<EOF
{"key": "value"}EOF
```

# command chaining

One powerful feature of this shell is to chain incoming messages to external applications. It works like the other unix shells:

This will pass through all incoming messages in topic **test/topic** to `grep`
```bash
sub test/topic | grep "Message"
```

## stderr forwarding

If you want to push stdout **and** stderr to the stdin of the next application:
```bash
sub test/topic | myExternalApplication |& grep "Message"
```

## long term applications

Normally the external applications will be started on **each incoming Message**. If you want to stream all incoming messages
to a single started application:
```bash
sub test/topic | grep "Message" &
```

## file redirection

If you want to write all incoming messages into files:
```bash
sub test/topic >> /tmp/test.msg
```

### only last incoming message

If you want to write only the latest incoming message to file:
```bash
sub test/topic > /tmp/last.msg
```

# Macros

Macros can be a list of commands which should be executed. Or it can be a more complex but more powerful script. 
Macros can have their own arguments. They can be defined in the environment file (`~/.config/mqtt-shell/my-env.yml`), 
the global settings (`~/.config/mqtt-shell/.global.yml`) or the global macro file (`~/.config/mqtt-shell/.macros.yml`)

## Macros - list of commands
```yaml
# ~/.config/mqtt-shell/.macros.yml

my-macro:
  description: Awesome description of my macro
  arguments:
    - message
  varargs: true
  commands:
    - pub test $1
```

Then you can use it in the mqtt-shell:
```bash
> sub test
> my-macro "Message#1" "Message#2"
test | Message#1
test | Message#2
```

## Macros - a complex script

The [golang text templating](https://pkg.go.dev/text/template) is used for the scripts. The macro arguments can be read
by **Arg** following by the **number of the argument**. So for the first argument: **Arg1** and for the second **Arg2** and
so on. 

Furthermore there are two custom functions available:

| name | argument | description | example |
|---|---|---|---|
| exec | &lt;cmdLine&gt; | Run the given command and return the result. You can also use pipes! | exec "date &#124; cut -d\  -f1" |
| log | &lt;format string&gt; [&lt;argument&gt;, ...] | Write the given content to the shell stdout. | log "Argument#1: %s" .Arg1 |

```yaml
# ~/.config/mqtt-shell/.macros.yml

my-macro:
  description: Awesome description of my macro
  arguments:
    - message
  varargs: true
  script: |-
    {{ log "Publish to topic" }}
    pub test {{ .Arg1 }} {{ exec "date" }}
```

Then you can use it in the mqtt-shell:
```bash
> sub test
> my-macro "Message#1" "Message#2"
test | Message#1 So 15. Aug 16:15:00 CEST 2021
test | Message#2 So 15. Aug 16:15:00 CEST 2021
```

# Color output

This shell is able to write colored output. Each time a new subscription is made, the messages for that subscription will have
a colored prefix. Each subscription (not topic!) should have an own color schema. Internally the shell will have a pool with
color codes. Each time a new subscription was made, the next color will get from that pool. After the pool is exhausted, 
the color choosing will start again. The color pool can be shown with the `color` command in the shell.

If you want to disable some colors, you have to put them in your yaml config file(s). Or use the option `-cb` for the current
session.

## Why chained applications will not show any color?

Because the mqtt-shell itself will start the chained application, the application can not detect if it operates on a tty. 
So normally the applications will think that their stdin is no tty. Most of the application have an option to force print
the color codes. For example grep:

```bash
sub test/topic | grep --color=always Message
```