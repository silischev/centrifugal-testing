package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"tests/common"
	"time"

	"github.com/FZambia/viper-lite"

	"github.com/centrifugal/centrifuge-go"
)

const (
	replicas = 1
)

var host string

func init() {
	host, _ = os.Hostname()
	rand.Seed(time.Now().UnixNano())
}

var defaultConfig = map[string]interface{}{
	"url":                              "ws://centrifugo:8000/connection/websocket",
	"message_format":                   "json",
	"clients_connections_number":       10050,
	"add_subscriber_delay_millisecond": 50,
}

type config struct {
	url                        string        // connection URI
	format                     string        // message format
	numClients                 int           // number of clients connections
	subscriberDelay            time.Duration // delay before adding new subscriber (millisecond)
	personalPublisherSlowDelay time.Duration // delay before adding new slow personal publisher (millisecond)
	personalPublisherFastDelay time.Duration // delay before adding new fast personal publisher (millisecond)
	groupPublisherSlowDelay    time.Duration // delay before adding new slow group publisher (millisecond)
	groupPublisherFastDelay    time.Duration // delay before adding new fast group publisher (millisecond)
	publishingDelay            time.Duration // delay after start slow publishers (second)
}

func getConfig() config {
	for k, v := range defaultConfig {
		viper.SetDefault(k, v)
	}

	viper.SetConfigFile("/tests/load/test_conf.json")
	err := viper.ReadInConfig()
	if err != nil {
		log.Println(err)
	}

	return config{
		url:                        viper.GetString("url"),
		format:                     viper.GetString("message_format"),
		numClients:                 viper.GetInt("clients_connections_number"),
		subscriberDelay:            viper.GetDuration("subscriber_delay_millisecond") * time.Millisecond,
		personalPublisherSlowDelay: viper.GetDuration("personal_publisher_slow_delay_millisecond") * time.Millisecond,
		personalPublisherFastDelay: viper.GetDuration("personal_publisher_fast_delay_millisecond") * time.Millisecond,
		groupPublisherSlowDelay:    viper.GetDuration("group_publisher_slow_delay_millisecond") * time.Millisecond,
		groupPublisherFastDelay:    viper.GetDuration("group_publisher_fast_delay_millisecond") * time.Millisecond,
		publishingDelay:            viper.GetDuration("publishing_delay_second") * time.Second,
	}
}

type pubEventHandler struct {
	Channel string
}

func (h *pubEventHandler) OnPublish(sub *centrifuge.Subscription, e centrifuge.PublishEvent) {
	var message Message
	err := json.Unmarshal(e.Data, &message)
	if err != nil {
		return
	}

	// @TODO
	if message.Host != host {
		// Only measure latency for messages born in this process.
		return
	}
	//statsdClient.Timing(fmt.Sprintf("messages_received.%s.latency", h.Channel), (time.Now().UnixNano()-message.Time)/1000/1000)
}

func main() {
	cfg := getConfig()

	go func() {
		for i := 0; i < cfg.numClients; i++ {
			time.Sleep(cfg.subscriberDelay)
			go func(i int) {
				runSubscriber(cfg)
			}(i)
		}
	}()

	go func() {
		// Run slow publisher from the beginning.
		go func() {
			runPersonalPublisher(cfg, cfg.personalPublisherSlowDelay)
		}()
		// Then wait to start publishing more.
		time.Sleep(cfg.publishingDelay)
		for i := 0; i < 10; i++ {
			go func() {
				runPersonalPublisher(cfg, cfg.personalPublisherFastDelay)
			}()
			sleepFor := rand.Intn(60) + 300
			time.Sleep(time.Duration(sleepFor) * time.Second)
		}
	}()

	go func() {
		// Run slow publisher from the beginning.
		go func() {
			runPersonalPublisher(cfg, cfg.personalPublisherSlowDelay)
		}()
		// Then wait to start publishing more.
		time.Sleep(30 * time.Minute)
		for i := 0; i < 10; i++ {
			go func() {
				runGroupPublisher(cfg, cfg.groupPublisherFastDelay)
			}()
			sleepFor := rand.Intn(60) + 300
			time.Sleep(time.Duration(sleepFor) * time.Second)
		}
	}()

	log.Fatalln(http.ListenAndServe(":8081", nil))
}

func personalChannel(numClients int) string {
	// While this is not truly "personal" channel as there will be
	// channel name collisions among maxBenchmarkClients this is ok
	// for our use case where we really just want a distribution close
	// to one unique channel per client and a way to publish messages
	// into personal channels.
	return "personal" + strconv.Itoa(rand.Intn(numClients*replicas*2))
}

func groupChannel(numClients int) string {
	// Each client will be subscribed to group channel. The amount of
	// subscribers in such group will be close to replica number of
	// client pods. For example if we have 100 clients pods to generate
	// maxBenchmarkClients connections then every group channel will
	// contain about 100 subscribers.
	return "group" + strconv.Itoa(rand.Intn(numClients*2))
}

func runSubscriber(cfg config) *centrifuge.Client {
	client := common.NewConnection(cfg.url, cfg.format)

	personalSub, err := client.NewSubscription(personalChannel(cfg.numClients))
	if err != nil {
		log.Fatalln(err)
	}

	personalSub.OnPublish(&pubEventHandler{"personal"})
	_ = personalSub.Subscribe()

	groupSub, err := client.NewSubscription(groupChannel(cfg.numClients))
	if err != nil {
		log.Fatalln(err)
	}

	groupSub.OnPublish(&pubEventHandler{"group"})
	_ = groupSub.Subscribe()

	err = client.Connect()
	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}

	go func() {
		// Periodically disconnect and connect back.
		for {
			min := 10
			max := 10 * 60
			ttlSeconds := rand.Intn(max-min) + min
			time.Sleep(time.Duration(ttlSeconds) * time.Second)
			err = client.Disconnect()
			if err != nil {
				log.Fatalf("Can't connect: %v\n", err)
			}

			time.Sleep(time.Second)
			err = client.Connect()
			if err != nil {
				log.Fatalf("Can't connect: %v\n", err)
			}
		}
	}()

	return client
}

// Message represents a message we publish into channels.
type Message struct {
	// Time is a current UNIX timestmap nanoseconds.
	Time int64 `json:"time"`
	// Host is a name of host where message was born.
	Host string `json:"host"`
}

func runPersonalPublisher(cfg config, sleep time.Duration) *centrifuge.Client {
	client := common.NewConnection(cfg.url, cfg.format)
	err := client.Connect()
	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}

	// Periodically publish messages into channels.
	go func() {
		for {
			// Publish into personal channel.
			time.Sleep(sleep)
			message := Message{
				Time: time.Now().UnixNano(),
				Host: host,
			}
			data, _ := json.Marshal(message)
			_, err := client.Publish(personalChannel(cfg.numClients), data)
			if err != nil {
				//statsdClient.Increment("publish.personal_channel.error")
			}
		}
	}()
	return client
}

func runGroupPublisher(cfg config, sleep time.Duration) *centrifuge.Client {
	client := common.NewConnection(cfg.url, cfg.format)
	err := client.Connect()
	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}

	// Periodically publish messages into channels.
	go func() {
		for {
			// Publish into group channel.
			time.Sleep(sleep)
			message := Message{
				Time: time.Now().UnixNano(),
				Host: host,
			}
			data, _ := json.Marshal(message)
			_, err := client.Publish(groupChannel(cfg.numClients), data)
			if err != nil {
				//statsdClient.Increment("publish.group_channel.error")
			}
		}
	}()
	return client
}
