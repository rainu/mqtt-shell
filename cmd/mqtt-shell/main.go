package main

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/rainu/mqtt-shell/internal/config"
	internalIo "github.com/rainu/mqtt-shell/internal/io"
	"io"
	"log"
)

func main() {
	var output io.Writer
	var mqttReconnectListener interface {
		OnMqttReconnect()
	}

	cfg := config.NewConfig()

	opts := MQTT.NewClientOptions()
	opts.AddBroker(*cfg.Broker)
	opts.SetClientID(*cfg.ClientId)
	if cfg.Username != nil {
		opts.SetUsername(*cfg.Username)
	}
	if cfg.Password != nil {
		opts.SetPassword(*cfg.Password)
	}
	opts.SetAutoReconnect(true)
	opts.SetCleanSession(*cfg.CleanSession)
	opts.SetOnConnectHandler(func(_ MQTT.Client) {
		if output != nil {
			output.Write([]byte("Successfully re-connected to mqtt broker.\n"))
		}
		if mqttReconnectListener != nil {
			mqttReconnectListener.OnMqttReconnect()
		}
	})
	opts.SetConnectionLostHandler(func(_ MQTT.Client, err error) {
		if output != nil {
			output.Write([]byte("Connection to broker lost. Reconnecting...\n"))
		}
	})

	client := MQTT.NewClient(opts)
	if t := client.Connect(); !t.Wait() || t.Error() != nil {
		log.Fatal(t.Error())
	}

	var subInformer interface {
		GetSubscriptions() []string
	}
	shell, err := internalIo.NewShell(func(s string) []string {
		return subInformer.GetSubscriptions()
	})
	if err != nil {
		log.Fatal(err)
	}
	output = shell
	inputChan := shell.Start()

	processor := internalIo.NewProcessor(shell, client)
	subInformer = processor
	mqttReconnectListener = processor

	//process loop
	processor.Process(inputChan)
}
