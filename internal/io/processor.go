package io

import (
	"bytes"
	"errors"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"io"
	"sort"
	"strconv"
	"sync"
)

type subscription struct {
	qos      byte
	callback mqtt.MessageHandler
}

type commandHandle struct {
	w         io.Closer
	closeChan chan interface{}
}

type processor struct {
	client mqtt.Client
	out    io.Writer

	longTermCommands map[string]commandHandle
	subscribedTopics map[string]subscription
}

func NewProcessor(out io.Writer, client mqtt.Client) *processor {
	return &processor{
		client:           client,
		out:              out,
		longTermCommands: map[string]commandHandle{},
		subscribedTopics: map[string]subscription{},
	}
}

func (p *processor) Process(input chan string) {
	for line := range input {
		chain, err := interpretLine(line)
		if err != nil {
			p.out.Write([]byte(err.Error() + "\n"))
			continue
		}
		err = p.handleCommand(chain)
		if err != nil {
			p.out.Write([]byte(err.Error() + "\n"))
		}
	}

	//close all long term chain inputs (will cause the normally exiting of underlying commands)
	for _, input := range p.longTermCommands {
		input.w.Close()
	}
}

func (p *processor) GetSubscriptions() []string {
	topics := make([]string, 0, len(p.subscribedTopics))
	for topic := range p.subscribedTopics {
		topics = append(topics, topic)
	}

	sort.Strings(topics)
	return topics
}

func (p *processor) HasSubscriptions() bool {
	return len(p.subscribedTopics) > 0
}

func (p *processor) OnMqttReconnect() {
	for topic, subscription := range p.subscribedTopics {
		p.client.Subscribe(topic, subscription.qos, subscription.callback)
	}
}

func (p *processor) handleCommand(chain Chain) error {
	if len(chain.Commands) == 0 {
		return nil
	}

	switch chain.Commands[0].Name {
	case commandHelp:
		return p.handleHelp(chain)
	case commandListColors:
		return p.handleColors(chain)
	case commandPub:
		return p.handlePub(chain)
	case commandSub:
		return p.handleSub(chain)
	case commandUnsub:
		return p.handleUnsub(chain)
	case commandList:
		return p.handleList(chain)
	default:
		return errors.New("unknown command")
	}
}

func (p *processor) handleHelp(chain Chain) error {
	p.out.Write([]byte(helpText))
	return nil
}

func (p *processor) handleColors(chain Chain) error {
	for _, dec := range decoratorPool {
		p.out.Write([]byte(fmt.Sprintf("%s\n", decorate(dec.String(), dec...))))
	}

	return nil
}

func (p *processor) handlePub(chain Chain) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%s\nUsage: "+commandPub+" [-r] [-q 0|1|2] <topic> <payload>", err.Error())
		}
	}()

	if len(chain.Commands[0].Arguments) < 2 {
		return errors.New("invalid arguments")
	}

	var topic, payload string
	qos := 0
	retained := false

	for i := 0; i < len(chain.Commands[0].Arguments); i++ {
		arg := chain.Commands[0].Arguments[i]

		switch arg {
		case "-r":
			retained = true
		case "-q":
			if i+1 < len(chain.Commands[0].Arguments) {
				var err error
				qos, err = strconv.Atoi(chain.Commands[0].Arguments[i+1])
				if err != nil {
					return fmt.Errorf("invalid qos level: %w", err)
				}
				if qos < 0 || qos > 3 {
					return errors.New("invalid qos level")
				}
				i++
			} else {
				return errors.New("invalid arguments")
			}
		default:
			if topic == "" {
				topic = arg
			} else if payload == "" {
				payload = arg
			} else {
				payload += " " + arg
			}
		}
	}

	if topic == "" || payload == "" {
		return errors.New("invalid arguments")
	}

	if token := p.client.Publish(topic, byte(qos), retained, payload); !token.Wait() {
		return token.Error()
	}
	return nil
}

func (p *processor) handleList(chain Chain) error {
	for _, topic := range p.GetSubscriptions() {
		p.out.Write([]byte(topic + "\n"))
	}
	return nil
}

func (p *processor) handleUnsub(chain Chain) error {
	for _, topic := range chain.Commands[0].Arguments {
		if ltWriter, ok := p.longTermCommands[topic]; ok {
			//close the command-input-stream (will end the underlying cmdchain)
			ltWriter.w.Close()
		}

		if token := p.client.Unsubscribe(topic); !token.Wait() {
			return token.Error()
		}
		delete(p.subscribedTopics, topic)
	}

	return nil
}

func (p *processor) handleSub(chain Chain) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%s\nUsage: "+commandSub+" [-q 0|1|2] <topic> [...topicN]", err.Error())
		}
	}()

	topics := make([]string, 0, 1)
	qos := 0

	for i := 0; i < len(chain.Commands[0].Arguments); i++ {
		arg := chain.Commands[0].Arguments[i]

		switch arg {
		case "-q":
			if i+1 < len(chain.Commands[0].Arguments) {
				var err error
				qos, err = strconv.Atoi(chain.Commands[0].Arguments[i+1])
				if err != nil {
					return fmt.Errorf("invalid qos level: %w", err)
				}
				if qos < 0 || qos > 3 {
					return errors.New("invalid qos level")
				}
				i++
			} else {
				return errors.New("invalid arguments")
			}
		default:
			topics = append(topics, arg)
		}
	}

	if len(topics) == 0 {
		return errors.New("invalid arguments")
	}

	for _, topic := range topics {
		clb, err := genSubHandler(p, topic, chain)
		if err != nil {
			return err
		}

		if token := p.client.Subscribe(topic, byte(qos), clb); !token.Wait() {
			return token.Error()
		}
		p.subscribedTopics[topic] = subscription{qos: byte(qos), callback: clb}
	}

	return nil
}

var genSubHandler = func(p *processor, topic string, chain Chain) (func(mqtt.Client, mqtt.Message), error) {
	if len(chain.Commands) == 1 {
		//the decorator will be saved because of inline func
		//so each message for the current sub have the same decorator
		decorators := getNextDecorator()

		return func(_ mqtt.Client, message mqtt.Message) {
			p.out.Write([]byte(decorate(message.Topic()+" |", decorators...) + " " + string(message.Payload()) + "\n"))
		}, nil
	}

	//long term chains with shell output work not very well together - so ignore this combination
	if chain.IsLongTerm() && chain.IsAppending() {
		return p.longTermSub(topic, chain)
	}

	//each new input will cause executing a new chain (short term)
	return p.shortTermSub(chain), nil
}

func (p *processor) longTermSub(topic string, chain Chain) (func(mqtt.Client, mqtt.Message), error) {
	//long term commands are commands which are running permanently in background
	//each new message will be written in ONE input pipe to that command
	r, w := io.Pipe()

	if prevWriter, ok := p.longTermCommands[topic]; ok {
		//close the previous command-input-stream
		prevWriter.w.Close()
	}
	p.longTermCommands[topic] = commandHandle{
		w:         w,
		closeChan: make(chan interface{}),
	}
	cmd, clb, err := chain.ToCommand(r)
	if err != nil {
		return nil, err
	}

	//start the chain in background
	go func() {
		defer r.Close()
		defer clb()
		defer close(p.longTermCommands[topic].closeChan)

		//the command chain will be finished if the underlying pipe is closed
		if err := cmd.Run(); err != nil {
			p.out.Write([]byte(err.Error() + "\n"))
		}
	}()

	return func(client mqtt.Client, message mqtt.Message) {
		//every time a new message will come, push them to the pipe of that chain
		w.Write(message.Payload())
		w.Write([]byte("\n"))
	}, nil
}

func (p *processor) shortTermSub(chain Chain) func(mqtt.Client, mqtt.Message) {
	//the decorator will be saved because of inline func
	//so each message for the current sub have the same decorator
	decorators := getNextDecorator()

	return func(client mqtt.Client, message mqtt.Message) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		writeError := func(err error) {
			p.out.Write([]byte(decorate(message.Topic()+" |", decorators...) + " " + err.Error() + "\n"))
		}

		go func() {
			defer wg.Done()

			writer := make([]io.Writer, 0, 1)
			if !chain.IsAppending() {
				writer = append(writer, &prefixWriter{
					Prefix:   decorate(message.Topic()+" |", decorators...) + " ",
					Delegate: p.out,
				})
			}

			cmd, clb, err := chain.ToCommand(bytes.NewReader(message.Payload()), writer...)
			defer clb()

			if err != nil {
				writeError(err)
				return
			}

			if err = cmd.Run(); err != nil {
				writeError(err)
			}
		}()

		wg.Wait()
	}
}
