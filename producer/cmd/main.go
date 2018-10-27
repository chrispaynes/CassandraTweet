package main

import (
	"github.com/gocql/gocql"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	ConsumerKey    string `split_words:"true"`
	ConsumerSecret string `split_words:"true"`
	AccessToken    string `split_words:"true"`
	AccessSecret   string `split_words:"true"`
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	var c Config

	err := envconfig.Process("TWITTER", &c)

	if err != nil {
		log.Fatal(err.Error())
	}

	// connects to Cassandra using docker container name as hostname
	cluster := gocql.NewCluster("cassandra")
	cluster.Keyspace = "tweeter"
	cluster.Consistency = gocql.Quorum
	session, err := cluster.CreateSession()

	if err != nil {
		log.Fatalf("failed to initialize Cassandra cluster connection: %s", err.Error())
	}

	defer session.Close()

	if err := session.Query(`insert into tweet(id, created_at, text) values (?, ?, ?)`,
		gocql.TimeUUID(), "Wed Aug 27 13:08:45 +0000 2008", "Hello World").Exec(); err != nil {
		log.Fatalf("failed to insert tweet into database: %s", err.Error())
	}
}
