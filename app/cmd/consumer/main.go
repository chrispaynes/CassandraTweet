package main

import (
	"encoding/json"
	"os"
	"os/signal"

	"github.com/ChimeraCoder/anaconda"
	kafka "github.com/Shopify/sarama"
	"github.com/gocql/gocql"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

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

	var cc CassandraConfig

	err := envconfig.Process("CASSANDRA", &cc)

	if err != nil {
		log.Fatalf("failed to load CassandraDB environment variables: %s", err.Error())
	}

	cluster := gocql.NewCluster(cc.Host...)
	cluster.Keyspace = cc.Keyspace
	cluster.Consistency = gocql.Quorum
	session, err := cluster.CreateSession()

	if err != nil {
		log.Fatalf("failed to initialize Cassandra cluster connection: %s", err.Error())
	}

	defer session.Close()

	consumer, err := kafka.NewConsumer([]string{"kafka1:9092"}, nil)

	if err != nil {
		log.Fatalf("failed to create new Kafka consumer: %s", err.Error())
	}

	defer func() {
		if err := consumer.Close(); err != nil {
			log.Fatalf("failed to close consumer: %s", err.Error())
		}
	}()

	partitionConsumer, err := consumer.ConsumePartition("Tweets", 0, kafka.OffsetNewest)

	if err != nil {
		log.Fatalf("failed to consume the kafka partition %s", err.Error())
	}

	defer func() {
		if err := partitionConsumer.Close(); err != nil {
			log.Fatalf("failed to close the kafka partition consumer: %s", err.Error())
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	consumed := 0

ConsumerLoop:
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			log.Infof("consuming message offset %d\n", msg.Offset)

			v := anaconda.Tweet{}

			err := json.Unmarshal(msg.Value, &v)

			if err != nil {
				log.Errorf("failed to unmarshalled consumed tweet: %s", err.Error())
			}

			consumed++

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

		case <-signals:
			break ConsumerLoop
		}
	}

}
