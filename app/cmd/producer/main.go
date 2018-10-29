package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ChimeraCoder/anaconda"
	kafka "github.com/Shopify/sarama"
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

	err := envconfig.Process("TWITTER", &tc)

	if err != nil {
		log.Fatal(err.Error())
	}

	anaconda.SetConsumerKey(tc.ConsumerKey)
	anaconda.SetConsumerSecret(tc.ConsumerSecret)
	api := anaconda.NewTwitterApi(tc.AccessToken, tc.AccessSecret)

	// v := url.Values{}
	_ = api
	// s := api.PublicStreamSample(v)
	s := anaconda.Stream{}
	defer s.Stop()

	config := kafka.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := kafka.NewAsyncProducer([]string{"kafka1:9092"}, config)
	if err != nil {
		log.Fatalf("failed to create Kafka producer: %s", err.Error())
	}

	log.Infof("successfully created new Kafka producer: %v", producer)

	// graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	var (
		wg                          sync.WaitGroup
		enqueued, successes, errors int
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range producer.Successes() {
			successes++
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range producer.Errors() {
			log.Printf("producer error: %s", err.Error())
			errors++
		}
	}()

	message := &kafka.ProducerMessage{Topic: "Tweets", Key: kafka.StringEncoder("kakfa1"), Value: kafka.StringEncoder("KAFKA TEST!!!")}
	log.Infof("created new Producer Message: %v", message)
	producer.Input() <- message
	enqueued++

	for t := range s.C {
		switch v := t.(type) {
		case anaconda.Tweet:
			if v.Lang == "en" && v.User.Id != 0 && v.User.CreatedAt != "" && v.User.Name != "" && v.User.ScreenName != "" && v.User.FollowersCount != 0 {
				go func() {
					// send tweet to Kafka topic

					var results []byte

					err := v.UnmarshalJSON(results)

					if err != nil {
						log.Errorf("could not unmarshall tweet to bytes: %s", err.Error())
					}

				ProducerLoop:
					for {
						message := &kafka.ProducerMessage{Topic: "tweets", Value: kafka.StringEncoder(string(results))}
						select {
						case producer.Input() <- message:
							enqueued++
						case <-signals:
							producer.AsyncClose() // shutdown the producer
							break ProducerLoop
						}
					}

					wg.Wait()

					log.Printf("produced: %d; errors: %d\n", successes, errors)

					// end kafka tweet logic
					if err != nil {
						log.Fatalf("failed to send twitter message to kafka stream: %s", err.Error())
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
