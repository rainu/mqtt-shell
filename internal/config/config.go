package config

import (
	"flag"
)

type Config struct {
	Broker       *string
	SubscribeQOS *int
	PublishQOS   *int
	Username     *string
	Password     *string
	ClientId     *string
	CleanSession *bool
}

func NewConfig() Config {
	cfg := Config{
		Broker:       flag.String("broker", "", "The broker URI. ex: tcp://127.0.0.1:1883"),
		SubscribeQOS: flag.Int("sub-qos", 0, "The default Quality of Service for subscription 0,1,2 (default 0)"),
		PublishQOS:   flag.Int("pub-qos", 1, "The default Quality of Service for publishing 0,1,2 (default 1)"),
		Username:     flag.String("user", "", "The User (optional)"),
		Password:     flag.String("password", "", "The password (optional)"),
		ClientId:     flag.String("client-id", "mqtt-shell", "The ClientID (optional)"),
		CleanSession: flag.Bool("clean-session", true, "By setting this flag, you are indicating that no messages saved by the broker for this client should be delivered."),
	}
	flag.Parse()

	return cfg
}
