package io

import "strings"

var helpText = `\u001b[7mPublishing a message\u001b[0m

  \u001b[1mpub [-r] [-q 0|1|2] <topic> <payload>\u001b[0m

    -r          retained
    -q [0|1|2]  QualityOfService (QoS) level

  \u001b[4mPublishing a multiline message\u001b[0m
  
    \u001b[1mpub my/topic <<EOF
  	  This is
  	  a multiline
  	  message
    EOF\u001b[0m

  \u001b[4mThis is also useful if you don't want to handle argument escaping\u001b[0m

    \u001b[1mpub my/topic <<EOF
    {"key": "value"}EOF\u001b[0m

\u001b[7mSubscribe to a topic\u001b[0m

  \u001b[1msub [-q 0|1|2] <topic> [...topicN]\u001b[0m

    -q [0|1|2]  QualityOfService (QoS) level

  \u001b[7mCommand chaining\u001b[0m
    One powerful feature of this shell is to chain incoming messages to external applications. 
    It works like the other unix shells.

    \u001b[4mThis will pass through all incoming messages in topic test/topic to grep\u001b[0m

      \u001b[1msub test/topic | grep "Message"\u001b[0m

    \u001b[4mIf you want to push stdout and stderr to the stdin of the next application:\u001b[0m

      \u001b[1msub test/topic | myExternalApplication |& grep "Message"\u001b[0m

    \u001b[4mNormally the external applications will be started on each incoming Message.\u001b[0m
    \u001b[4mIf you want to stream all incoming messages to a single started application:\u001b[0m

      \u001b[1msub test/topic | grep "Message" &\u001b[0m

    \u001b[4mIf you want to write all incoming messages into files:\u001b[0m

      \u001b[1msub test/topic >> /tmp/test.msg\u001b[0m

\u001b[7mUnsubscribe a topic\u001b[0m

  \u001b[1munsub <topic> [...topicN]\u001b[0m

\u001b[7mList all available commands\u001b[0m

  \u001b[1m.ls\u001b[0m

\u001b[7mList all available macros\u001b[0m

  \u001b[1m.macro\u001b[0m

\u001b[7mList all available colors schemas\u001b[0m

  \u001b[1m.lsc\u001b[0m

\u001b[7mExit the shell\u001b[0m

  \u001b[1mexit\u001b[0m

For more information see https://github.com/rainu/mqtt-shell
`

func init() {
	helpText = strings.Replace(helpText, `\u001b`, "\u001b", -1)
}
