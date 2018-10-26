package main

import (
	"github.com/kelseyhightower/envconfig"
	"log"
)

type Config struct {
	ConsumerKey    string `split_words:"true"`
	ConsumerSecret string `split_words:"true"`
	AccessToken    string `split_words:"true"`
	AccessSecret   string `split_words:"true"`
}

func main() {
	var c Config

	err := envconfig.Process("TWITTER", &c)

	if err != nil {
		log.Fatal(err.Error())
	}
}
