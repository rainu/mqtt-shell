package main

import (
	"crypto/tls"
	"crypto/x509"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/rainu/mqtt-shell/internal/config"
	internalIo "github.com/rainu/mqtt-shell/internal/io"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var mqttReconnectListener interface {
	OnMqttReconnect()
}

func main() {
	cfg := config.NewConfig()
	interactive := !cfg.NonInteractive

	mqttClient := establishMqtt(cfg)

	var output io.Writer
	var inputChan chan string
	var subInformer interface {
		GetSubscriptions() []string
	}
	signals := make(chan os.Signal, 1)

	if interactive {
		shell, err := internalIo.NewShell(cfg.Prompt, cfg.HistoryFile, cfg.Macros, func(s string) []string {
			return subInformer.GetSubscriptions()
		})
		if err != nil {
			log.Fatal(err)
		}
		output = shell
		inputChan = shell.Start()
	} else {
		//non interactive mean that there is no shell open
		inputChan = make(chan string)
		output = os.Stdout

		//reacting to signals (interrupt)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	}

	//execute the start commands
	go func() {
		for _, command := range cfg.StartCommands {
			inputChan <- command
		}

		if cfg.NonInteractive {
			close(inputChan)
		}
	}()

	processor := internalIo.NewProcessor(output, mqttClient)
	subInformer = processor
	mqttReconnectListener = processor

	//process loop
	processor.Process(inputChan)

	if !interactive && processor.HasSubscriptions() {
		//wait for interrupt
		<-signals
	}
}

func establishMqtt(cfg config.Config) MQTT.Client {
	opts := MQTT.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetClientID(cfg.ClientId)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}

	if cfg.CaFile != "" {
		certPool := x509.NewCertPool()
		certFile, err := ioutil.ReadFile(cfg.CaFile)
		if err != nil {
			log.Fatal(err)
		}
		ok := certPool.AppendCertsFromPEM(certFile)
		if !ok {
			log.Fatal("Failed to parse ca certificate!")
		}

		opts.SetTLSConfig(&tls.Config{
			RootCAs: certPool,
		})
	}

	opts.SetAutoReconnect(true)
	opts.SetCleanSession(cfg.CleanSession)

	firstConnect := true
	opts.SetOnConnectHandler(func(_ MQTT.Client) {
		if firstConnect {
			println("Successfully connected to mqtt broker.")
		} else {
			println("Successfully re-connected to mqtt broker.")
		}
		if mqttReconnectListener != nil {
			mqttReconnectListener.OnMqttReconnect()
		}

		firstConnect = false
	})
	opts.SetConnectionLostHandler(func(_ MQTT.Client, err error) {
		println("Connection to broker lost. Reconnecting...")
	})

	client := MQTT.NewClient(opts)
	if t := client.Connect(); !t.Wait() || t.Error() != nil {
		log.Fatal(t.Error())
	}
	return client
}
