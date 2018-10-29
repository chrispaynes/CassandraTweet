package main

import (
	"encoding/json"
	"net/url"
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

	v := url.Values{}
	s := api.PublicStreamSample(v)
	defer s.Stop()

	config := kafka.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := kafka.NewAsyncProducer([]string{"kafka1:9092"}, config)
	if err != nil {
		log.Fatalf("failed to create Kafka producer: %s", err.Error())
	}

	log.Infof("successfully created new Kafka producer: %v", producer)
	log.Infof("twitter stream: %v", s)

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

	for t := range s.C {
		switch v := t.(type) {
		case anaconda.Tweet:
			if v.Lang == "en" && v.User.Id != 0 && v.User.CreatedAt != "" && v.User.Name != "" && v.User.ScreenName != "" && v.User.FollowersCount != 0 {
				go func() {
					b, err := json.Marshal(v)

					if err != nil {
						log.Fatalf("could not unmarshal tweet to bytes: %s", err.Error())
					}

					message := &kafka.ProducerMessage{Topic: "Tweets", Key: kafka.StringEncoder("kafka1"), Value: kafka.ByteEncoder(b)}
					log.Infof("created new Producer Message: %v", message)

					select {
					case producer.Input() <- message:
						enqueued++
					case <-signals:
						producer.AsyncClose() // shutdown the producer
						break
					}

					wg.Wait()

					log.Printf("produced: %d; errors: %d\n", successes, errors)

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
