package io

import (
	"bytes"
	"errors"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/mock/gomock"
	mock_io "github.com/rainu/mqtt-shell/internal/io/mocks"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
	"time"
)

//go:generate mockgen -destination=mocks/mqttClient.go -package=mock_io github.com/eclipse/paho.mqtt.golang Client
//go:generate mockgen -destination=mocks/mqttMessage.go -package=mock_io github.com/eclipse/paho.mqtt.golang Message
//go:generate mockgen -destination=mocks/mqttToken.go -package=mock_io github.com/eclipse/paho.mqtt.golang Token
//go:generate mockgen -destination=mocks/ioWriter.go -package=mock_io io Closer

func TestProcessor_Process_interpretError(t *testing.T) {
	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		assert.Equal(t, "<inputLine>", line)
		return Chain{}, errors.New("someError")
	}

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "someError\n", output.String())
}

func TestProcessor_Process_unknownCommand(t *testing.T) {
	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: "UNKOWN"}}}, nil
	}

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "unknown command\n", output.String())
}

func TestProcessor_Process_noCommands(t *testing.T) {
	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{}, nil
	}

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "", output.String())
}

func TestProcessor_Process_helpCommand(t *testing.T) {
	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandHelp}}}, nil
	}

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, helpText, output.String())
}

func TestProcessor_Process_listColorCommand(t *testing.T) {
	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	odp := decoratorPool
	defer func() {
		decoratorPool = odp
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandListColors}}}, nil
	}
	decoratorPool = []decorator{{"32"}}

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "\x1b[32m32\x1b[0m\n", output.String())
}

func TestProcessor_Process_listCommand(t *testing.T) {
	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandList}}}, nil
	}

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)
	toTest.subscribedTopics["a/topic"] = subscription{}
	toTest.subscribedTopics["b/topic"] = subscription{}
	toTest.subscribedTopics["c/topic"] = subscription{}

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "a/topic\nb/topic\nc/topic\n", output.String())
}

func TestProcessor_Process_unsubCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandUnsub, Arguments: []string{"a/topic"}}}}, nil
	}

	mockToken := mock_io.NewMockToken(ctrl)
	mockToken.EXPECT().Wait().Return(true)
	mockMqtt := mock_io.NewMockClient(ctrl)
	mockMqtt.EXPECT().Unsubscribe(gomock.Eq("a/topic")).Return(mockToken)

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, mockMqtt)
	toTest.subscribedTopics["a/topic"] = subscription{}
	toTest.subscribedTopics["b/topic"] = subscription{}

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "", output.String())

	_, exists := toTest.subscribedTopics["a/topic"]
	assert.False(t, exists)
}

func TestProcessor_Process_unsubCommand_errorWhileUnsub(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandUnsub, Arguments: []string{"a/topic"}}}}, nil
	}

	mockToken := mock_io.NewMockToken(ctrl)
	mockToken.EXPECT().Wait().Return(false)
	mockToken.EXPECT().Error().Return(errors.New("someError"))
	mockMqtt := mock_io.NewMockClient(ctrl)
	mockMqtt.EXPECT().Unsubscribe(gomock.Eq("a/topic")).Return(mockToken)

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, mockMqtt)
	toTest.subscribedTopics["a/topic"] = subscription{}
	toTest.subscribedTopics["b/topic"] = subscription{}

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "someError\n", output.String())

	_, exists := toTest.subscribedTopics["a/topic"]
	assert.True(t, exists)
}

func TestProcessor_Process_pubCommand_insufficientArguments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandPub, Arguments: []string{}}}}, nil
	}

	mockMqtt := mock_io.NewMockClient(ctrl)

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, mockMqtt)

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "invalid arguments\nUsage: pub [-r] [-q 0|1|2] <topic> <payload>\n", output.String())
}

func TestProcessor_Process_pubCommand_invalidQoS(t *testing.T) {
	tests := []struct {
		qos      string
		expected string
	}{
		{"NAN", "invalid qos level: strconv.Atoi: parsing \"NAN\": invalid syntax\nUsage: pub [-r] [-q 0|1|2] <topic> <payload>\n"},
		{"-1", "invalid qos level\nUsage: pub [-r] [-q 0|1|2] <topic> <payload>\n"},
		{"4", "invalid qos level\nUsage: pub [-r] [-q 0|1|2] <topic> <payload>\n"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("TestProcessor_Process_pubCommand_invalidQoS_%d", i), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			oil := interpretLine
			defer func() {
				interpretLine = oil
			}()

			interpretLine = func(line string) (Chain, error) {
				return Chain{Commands: []Command{{Name: commandPub, Arguments: []string{"-q", test.qos, "test/topic", "PAYLOAD"}}}}, nil
			}

			mockMqtt := mock_io.NewMockClient(ctrl)

			output := &bytes.Buffer{}
			toTest := NewProcessor(output, mockMqtt)

			toTest.Process(filledChan("<inputLine>"))

			assert.Equal(t, test.expected, output.String())
		})
	}
}

func TestProcessor_Process_pubCommand_missingPayload(t *testing.T) {
	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandPub, Arguments: []string{"-r", "-q", "1"}}}}, nil
	}

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "invalid arguments\nUsage: pub [-r] [-q 0|1|2] <topic> <payload>\n", output.String())
}

func TestProcessor_Process_pubCommand_errorOnPublishing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandPub, Arguments: []string{"-r", "-q", "1", "test/topic", "PAY", "LOAD"}}}}, nil
	}

	mockToken := mock_io.NewMockToken(ctrl)
	mockToken.EXPECT().Wait().Return(false)
	mockToken.EXPECT().Error().Return(errors.New("someError"))
	mockMqtt := mock_io.NewMockClient(ctrl)
	mockMqtt.EXPECT().Publish(gomock.Eq("test/topic"), gomock.Eq(byte(1)), gomock.Eq(true), gomock.Eq("PAY LOAD")).Return(mockToken)

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, mockMqtt)

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "someError\nUsage: pub [-r] [-q 0|1|2] <topic> <payload>\n", output.String())
}

func TestProcessor_Process_pubCommand_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandPub, Arguments: []string{"-r", "-q", "1", "test/topic", "PAY", "LOAD"}}}}, nil
	}

	mockToken := mock_io.NewMockToken(ctrl)
	mockToken.EXPECT().Wait().Return(true)
	mockMqtt := mock_io.NewMockClient(ctrl)
	mockMqtt.EXPECT().Publish(gomock.Eq("test/topic"), gomock.Eq(byte(1)), gomock.Eq(true), gomock.Eq("PAY LOAD")).Return(mockToken)

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, mockMqtt)

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "", output.String())
}

func TestProcessor_Process_subCommand_invalidArguments(t *testing.T) {
	tests := []struct {
		args     []string
		expected string
	}{
		{[]string{}, "invalid arguments\nUsage: sub [-q 0|1|2] <topic> [...topicN]\n"},
		{[]string{"-q"}, "invalid arguments\nUsage: sub [-q 0|1|2] <topic> [...topicN]\n"},
		{[]string{"-q", "NAN"}, "invalid qos level: strconv.Atoi: parsing \"NAN\": invalid syntax\nUsage: sub [-q 0|1|2] <topic> [...topicN]\n"},
		{[]string{"-q", "-1"}, "invalid qos level\nUsage: sub [-q 0|1|2] <topic> [...topicN]\n"},
		{[]string{"-q", "4"}, "invalid qos level\nUsage: sub [-q 0|1|2] <topic> [...topicN]\n"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("TestProcessor_Process_subCommand_invalidArguments_%d", i), func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			oil := interpretLine
			defer func() {
				interpretLine = oil
			}()

			ogsh := genSubHandler
			defer func() {
				genSubHandler = ogsh
			}()

			interpretLine = func(line string) (Chain, error) {
				return Chain{Commands: []Command{{Name: commandSub, Arguments: test.args}}}, nil
			}

			mockMqtt := mock_io.NewMockClient(ctrl)

			output := &bytes.Buffer{}
			toTest := NewProcessor(output, mockMqtt)

			genSubHandler = func(p *processor, topic string, chain Chain) (func(mqtt.Client, mqtt.Message), error) {
				assert.Same(t, toTest, p)
				assert.Equal(t, "test/topic", topic)

				testChain, _ := interpretLine("")
				assert.Equal(t, testChain, chain)

				return func(mqtt.Client, mqtt.Message) {}, nil
			}

			toTest.Process(filledChan("<inputLine>"))

			assert.Equal(t, test.expected, output.String())
		})
	}
}

func TestProcessor_Process_subCommand_errorOnSubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	ogsh := genSubHandler
	defer func() {
		genSubHandler = ogsh
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandSub, Arguments: []string{"-q", "1", "test/topic"}}}}, nil
	}

	mockToken := mock_io.NewMockToken(ctrl)
	mockToken.EXPECT().Wait().Return(false)
	mockToken.EXPECT().Error().Return(errors.New("someError"))
	mockMqtt := mock_io.NewMockClient(ctrl)
	mockMqtt.EXPECT().Subscribe(gomock.Eq("test/topic"), gomock.Eq(byte(1)), gomock.Any()).Return(mockToken)

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, mockMqtt)

	genSubHandler = func(p *processor, topic string, chain Chain) (func(mqtt.Client, mqtt.Message), error) {
		assert.Same(t, toTest, p)
		assert.Equal(t, "test/topic", topic)

		testChain, _ := interpretLine("")
		assert.Equal(t, testChain, chain)

		return func(mqtt.Client, mqtt.Message) {}, nil
	}

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "someError\nUsage: sub [-q 0|1|2] <topic> [...topicN]\n", output.String())
}

func TestProcessor_Process_subCommand_genSub(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	ogsh := genSubHandler
	defer func() {
		genSubHandler = ogsh
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandSub, Arguments: []string{"-q", "1", "test/topic"}}}}, nil
	}
	mockMqtt := mock_io.NewMockClient(ctrl)

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, mockMqtt)

	genSubHandler = func(p *processor, topic string, chain Chain) (func(mqtt.Client, mqtt.Message), error) {
		assert.Same(t, toTest, p)
		assert.Equal(t, "test/topic", topic)

		testChain, _ := interpretLine("")
		assert.Equal(t, testChain, chain)

		return nil, errors.New("someError")
	}

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "someError\nUsage: sub [-q 0|1|2] <topic> [...topicN]\n", output.String())
}

func TestProcessor_Process_subCommand_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oil := interpretLine
	defer func() {
		interpretLine = oil
	}()

	ogsh := genSubHandler
	defer func() {
		genSubHandler = ogsh
	}()

	interpretLine = func(line string) (Chain, error) {
		return Chain{Commands: []Command{{Name: commandSub, Arguments: []string{"-q", "1", "test/topic"}}}}, nil
	}

	mockToken := mock_io.NewMockToken(ctrl)
	mockToken.EXPECT().Wait().Return(true)
	mockMqtt := mock_io.NewMockClient(ctrl)
	mockMqtt.EXPECT().Subscribe(gomock.Eq("test/topic"), gomock.Eq(byte(1)), gomock.Any()).Return(mockToken)

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, mockMqtt)

	genSubHandler = func(p *processor, topic string, chain Chain) (func(mqtt.Client, mqtt.Message), error) {
		assert.Same(t, toTest, p)
		assert.Equal(t, "test/topic", topic)

		testChain, _ := interpretLine("")
		assert.Equal(t, testChain, chain)

		return func(mqtt.Client, mqtt.Message) {}, nil
	}

	toTest.Process(filledChan("<inputLine>"))

	assert.Equal(t, "", output.String())
	assert.Equal(t, byte(1), toTest.subscribedTopics["test/topic"].qos)
	assert.NotNil(t, toTest.subscribedTopics["test/topic"].callback)
}

func TestGenSubHandler_simple(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ognd := getNextDecorators
	defer func() {
		getNextDecorators = ognd
	}()
	getNextDecorators = func() decorator {
		return []string{"1"}
	}

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	fn, err := genSubHandler(toTest, "a/topic", Chain{Commands: []Command{{Name: "sub"}}})
	assert.NoError(t, err)

	//call the generated handler and see what he does
	testMessage := mock_io.NewMockMessage(ctrl)
	testMessage.EXPECT().Topic().Return("a/topic")
	testMessage.EXPECT().Payload().Return([]byte("PAYLOAD"))
	fn(nil, testMessage)

	assert.Equal(t, "\x1b[1ma/topic |\x1b[0m PAYLOAD\n", output.String())
}

func TestGenSubHandler_longTermSub(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	outputFile := path.Join(os.TempDir(), "longTermSub.txt")
	defer os.Remove(outputFile)

	testChain, err := interpretLine(fmt.Sprintf(`%s a/topic | grep "a" > %s &`, commandSub, outputFile))
	assert.NoError(t, err)

	mockCloser := mock_io.NewMockCloser(ctrl)
	mockCloser.EXPECT().Close()
	toTest.longTermCommands["a/topic"] = commandHandle{w: mockCloser}

	fn, err := genSubHandler(toTest, "a/topic", testChain)
	assert.NoError(t, err)

	//call the generated handler and see what he does
	testMessage := mock_io.NewMockMessage(ctrl)
	firstCall := testMessage.EXPECT().Payload().Return([]byte("PAYLOAD"))
	testMessage.EXPECT().Payload().After(firstCall).Return([]byte("payload"))
	fn(nil, testMessage)
	fn(nil, testMessage)

	cmdHandle, exists := toTest.longTermCommands["a/topic"]
	assert.True(t, exists)
	assert.NoError(t, cmdHandle.w.Close())

	// wait until command is finished
	select {
	case <-time.After(1 * time.Second):
		assert.Fail(t, "timout reached while waiting for command to end")
	case <-cmdHandle.closeChan:
	}

	fileContent, err := os.ReadFile(outputFile)
	assert.NoError(t, err)

	assert.Equal(t, "payload\n", string(fileContent))
	assert.Equal(t, "", output.String())
}

func TestGenSubHandler_longTermSub_invalidCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	outputFile := path.Join(os.TempDir(), "longTermSub.txt")
	defer os.Remove(outputFile)

	testChain, err := interpretLine(fmt.Sprintf(`%s a/topic | iNvAlIdC0mManD "a" > %s &`, commandSub, outputFile))
	assert.NoError(t, err)

	fn, err := genSubHandler(toTest, "a/topic", testChain)
	assert.NoError(t, err)

	//call the generated handler and see what he does
	testMessage := mock_io.NewMockMessage(ctrl)
	firstCall := testMessage.EXPECT().Payload().Return([]byte("PAYLOAD"))
	testMessage.EXPECT().Payload().After(firstCall).Return([]byte("payload"))
	fn(nil, testMessage)
	fn(nil, testMessage)

	cmdHandle, exists := toTest.longTermCommands["a/topic"]
	assert.True(t, exists)
	assert.NoError(t, cmdHandle.w.Close())

	// wait until command is finished
	select {
	case <-time.After(1 * time.Second):
		assert.Fail(t, "timout reached while waiting for command to end")
	case <-cmdHandle.closeChan:
	}

	fileContent, err := os.ReadFile(outputFile)
	assert.NoError(t, err)

	assert.Equal(t, "", string(fileContent))
	assert.Equal(t, "failed to start command: exec: \"iNvAlIdC0mManD\": executable file not found in $PATH\n", output.String())
}

func TestGenSubHandler_shortTermSub(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ognd := getNextDecorators
	defer func() {
		getNextDecorators = ognd
	}()
	getNextDecorators = func() decorator {
		return []string{"1"}
	}

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	testChain, err := interpretLine(fmt.Sprintf(`%s a/topic | grep "a"`, commandSub))
	assert.NoError(t, err)

	fn, err := genSubHandler(toTest, "a/topic", testChain)
	assert.NoError(t, err)

	//call the generated handler and see what he does
	testMessage := mock_io.NewMockMessage(ctrl)
	testMessage.EXPECT().Topic().Return("a/topic").AnyTimes()
	firstCall := testMessage.EXPECT().Payload().Return([]byte("PAYLOAD"))
	testMessage.EXPECT().Payload().After(firstCall).Return([]byte("payload"))
	fn(nil, testMessage)
	fn(nil, testMessage)

	assert.Equal(t, "\x1b[1ma/topic |\x1b[0m payload\n", output.String(), `only the "little" payload should be matched (grep)`)
}

func TestGenSubHandler_shortTermSub_invalidCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ognd := getNextDecorators
	defer func() {
		getNextDecorators = ognd
	}()
	getNextDecorators = func() decorator {
		return []string{"1"}
	}

	output := &bytes.Buffer{}
	toTest := NewProcessor(output, nil)

	testChain, err := interpretLine(fmt.Sprintf(`%s a/topic | iNvAlIdC0mManD "a"`, commandSub))
	assert.NoError(t, err)

	fn, err := genSubHandler(toTest, "a/topic", testChain)
	assert.NoError(t, err)

	//call the generated handler and see what he does
	testMessage := mock_io.NewMockMessage(ctrl)
	testMessage.EXPECT().Topic().Return("a/topic").AnyTimes()
	testMessage.EXPECT().Payload().Return([]byte("PAYLOAD"))
	fn(nil, testMessage)

	assert.Equal(t, "\x1b[1ma/topic |\x1b[0m failed to start command: exec: \"iNvAlIdC0mManD\": executable file not found in $PATH\n", output.String())
}

func filledChan(content ...string) chan string {
	result := make(chan string, len(content))
	for _, c := range content {
		result <- c
	}
	close(result)

	return result
}
