package config

import "strings"

var helpText = `\u001b[7mSetting files\u001b[0m
  All options can be written in separate environment files (one per environment) or for global settings 
  in the \u001b[1m.global.yml\u001b[0m file. These files must be stores inside the shel-config directory (\u001b[1m~/.mqtt-shell\u001b[0m).

\u001b[7mEnvironment configurations\u001b[0m
  You can create yaml files where you can configure predefined configuration. This can be helpful for 
  different mqtt environments. This files must be stored in the environment directory (by default \u001b[1m~/.mqtt-shell\u001b[0m).

  For example (\u001b[1mexample.yml\u001b[0m):
  
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
    color-blacklist:
      - "38;5;237"

  \u001b[4m$ ./mqtt-shell -e example\u001b[0m

\u001b[7mMacros\u001b[0m
  Macros can be a list of commands which should be executed. Or it can be a more complex but more powerful script. 
  Macros can have their own arguments. They can be defined in the environment file (\u001b[1m~/.mqtt-shell/my-env.yml\u001b[0m), the 
  global settings (\u001b[1m~/.mqtt-shell/.global.yml\u001b[0m) or the global macro file (\u001b[1m~/.mqtt-shell/.macros.yml\u001b[0m)

\u001b[7mMacros - list of commands\u001b[0m

  # ~/.mqtt-shell/.macros.yml
  my-macro:
    description: Awesome description of my macro
    arguments:
      - message
    varargs: true
    commands:
      - pub test $1

  Then you can use it in the mqtt-shell:
  
    > sub test
    > my-macro "Message#1" "Message#2"
    test | Message#1
    test | Message#2

\u001b[7mMacros - a complex script\u001b[0m

  The \u001b[4mgolang text templating\u001b[0m(https://pkg.go.dev/text/template) is used for the scripts. 
  The macro arguments can be read by \u001b[1mArg\u001b[0m following by the \u001b[1mnumber of the argument\u001b[0m.
  So for the first argument: \u001b[1mArg1\u001b[0m and for the second \u001b[1mArg2\u001b[0m and so on.

  Furthermore there are two custom functions available:

  +---------------------------------------------------------------------------------------------------------------------+
  | \u001b[4mname\u001b[0m | \u001b[4margument\u001b[0m                          | \u001b[4mdescription\u001b[0m                                  | \u001b[4mexample\u001b[0m                   |
  +---------------------------------------------------------------------------------------------------------------------+
  | exec | <cmdLine>                         | Run the given command and return the result. | exec "date cut -d\ -f1"   |
  |      |                                   | You can also use pipes!                      |                           |
  +---------------------------------------------------------------------------------------------------------------------+
  | log  | <format string> [<argument>, ...] | Write the given content to the shell stdout. | log "Argument#1: %s" .Arg1|
  +---------------------------------------------------------------------------------------------------------------------+

  # ~/.mqtt-shell/.macros.yml
  my-macro:
    description: Awesome description of my macro
    arguments:
      - message
    varargs: true
    script: |-
      {{ log "Publish to topic" }}
      pub test {{ .Arg1 }} {{ exec "date" }}

  Then you can use it in the mqtt-shell:

    > sub test
    > my-macro "Message#1" "Message#2"
    test | Message#1 So 15. Aug 16:15:00 CEST 2021
    test | Message#2 So 15. Aug 16:15:00 CEST 2021

For more information see https://github.com/rainu/mqtt-shell
`

func init() {
	helpText = strings.Replace(helpText, `\u001b`, "\u001b", -1)
}
