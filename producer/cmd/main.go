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

// user represents the author of a tweet and a CassandraDB User-Defined-Type(UDT)
type user struct {
	UserID     int64  `cql:"user_id"`
	CreatedAt  string `cql:"created_at"`
	Name       string `cql:"name"`
	ScreenName string `cql:"screen_name"`
	Followers  int    `cql:"followers"`
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
	defer s.Stop()

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
			if v.Lang == "en" && v.User.Id != 0 && v.User.CreatedAt != "" && v.User.Name != "" && v.User.ScreenName != "" && v.User.FollowersCount != 0 {
				go func() {
					err := session.Query(`insert into tweet(id, user, created_at, text) values (?, ?, ?, ?)`,
						gocql.TimeUUID(),
						user{UserID: v.User.Id, CreatedAt: v.User.CreatedAt, Name: v.User.Name, ScreenName: v.User.ScreenName, Followers: v.User.FollowersCount},
						v.CreatedAt,
						v.Text).Exec()

					if err != nil {
						log.Fatalf("failed to insert tweet into database: %s", err.Error())
					}
				}()
			}
		}
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	log.Info("stopping stream")
	s.Stop()
}
