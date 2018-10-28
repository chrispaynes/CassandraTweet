package main

import (
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChimeraCoder/anaconda"
	"github.com/gocql/gocql"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

// TwitterConfig represents a connection to the Twitter API
type TwitterConfig struct {
	ConsumerKey    string `split_words:"true"`
	ConsumerSecret string `split_words:"true"`
	AccessToken    string `split_words:"true"`
	AccessSecret   string `split_words:"true"`
}

// CassandraConfig represents a connection to the Cassandra DB cluster
type CassandraConfig struct {
	Keyspace   string   `split_words:"true"`
	TweetTable string   `split_words:"true"`
	Host       []string `split_words:"true"`
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	var tc TwitterConfig
	var cc CassandraConfig

	err := envconfig.Process("TWITTER", &tc)

	if err != nil {
		log.Fatal(err.Error())
	}

	anaconda.SetConsumerKey(tc.ConsumerKey)
	anaconda.SetConsumerSecret(tc.ConsumerSecret)
	api := anaconda.NewTwitterApi(tc.AccessToken, tc.AccessSecret)

	v := url.Values{}
	s := api.PublicStreamSample(v)

	err = envconfig.Process("CASSANDRA", &cc)

	if err != nil {
		log.Fatal(err.Error())
	}

	cluster := gocql.NewCluster(cc.Host...)
	cluster.Keyspace = cc.Keyspace
	cluster.Consistency = gocql.Quorum
	session, err := cluster.CreateSession()

	if err != nil {
		log.Fatalf("failed to initialize Cassandra cluster connection: %s", err.Error())
	}

	defer session.Close()

	for t := range s.C {
		switch v := t.(type) {
		case anaconda.Tweet:
			fmt.Printf("%v: %-15s: %s\n", v.CreatedAt, v.User.ScreenName, v.Text)
			if err := session.Query(`insert into tweet(id, created_at, text) values (?, ?, ?)`,
				gocql.TimeUUID(), v.CreatedAt, v.Text).Exec(); err != nil {
				log.Fatalf("failed to insert tweet into database: %s", err.Error())
			}
		}
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	log.Info("stopping stream")
	s.Stop()
}
