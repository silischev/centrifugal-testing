package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/centrifugal/centrifuge-go"
	"gopkg.in/alexcesaro/statsd.v2"
)

const numClientConnections = 10050

var host string

func init() {
	host, _ = os.Hostname()
	rand.Seed(time.Now().UnixNano())
}

var statsdClient *statsd.Client

type eventHandler struct {
	Channel string
}

func (h *eventHandler) OnPublish(sub *centrifuge.Subscription, e centrifuge.PublishEvent) {
	var message Message
	err := json.Unmarshal(e.Data, &message)
	if err != nil {
		return
	}
	statsdClient.Increment(fmt.Sprintf("messages_received.%s.total", h.Channel))
	if message.Host != host {
		// Only measure latency for messages born in this process.
		return
	}
	statsdClient.Timing(fmt.Sprintf("messages_received.%s.latency", h.Channel), (time.Now().UnixNano()-message.Time)/1000/1000)
}

func main() {

	cfg := config.NewDefaultConfig()
	err := cfg.Load(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatal(err)
	}

	statsdClient, err = NewStatsd(
		cfg.Statsd.Host,
		cfg.Statsd.Port,
		cfg.Statsd.Protocol,
		cfg.Statsd.Prefix,
		cfg.Statsd.Enable,
	)
	if err != nil {
		log.Fatalf("statsd client error: %v", err)
	}

	url := cfg.URL
	if url == "" {
		log.Fatal("no server url provided in config")
	}

	var clientsMu sync.Mutex
	clients := []*centrifuge.Client{}
	numClients := numClientConnections
	go func() {
		for i := 0; i < numClients; i++ {
			time.Sleep(50 * time.Millisecond)
			go func(i int) {
				client := runSubscriber(cfg, url, i)
				clientsMu.Lock()
				clients = append(clients, client)
				clientsMu.Unlock()
			}(i)
		}
	}()

	go func() {
		// Run slow publisher from the beginning.
		go func() {
			runPersonalPublisher(cfg, url, 60*time.Second)
		}()
		// Then wait to start publishing more.
		time.Sleep(30 * time.Minute)
		for i := 0; i < 10; i++ {
			go func() {
				runPersonalPublisher(cfg, url, 100*time.Millisecond)
			}()
			sleepFor := rand.Intn(60) + 300
			time.Sleep(time.Duration(sleepFor) * time.Second)
		}
	}()

	go func() {
		// Run slow publisher from the beginning.
		go func() {
			runPersonalPublisher(cfg, url, 60*time.Second)
		}()
		// Then wait to start publishing more.
		time.Sleep(30 * time.Minute)
		for i := 0; i < 10; i++ {
			go func() {
				runGroupPublisher(cfg, url, 100*time.Millisecond)
			}()
			sleepFor := rand.Intn(60) + 300
			time.Sleep(time.Duration(sleepFor) * time.Second)
		}
	}()

	http.HandleFunc("/_info", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("ok"))
	})

	go func() {
		if err := http.ListenAndServe(":8890", nil); err != nil {
			panic(err)
		}
	}()

	waitExitSignal(clients)
	fmt.Println("exiting")
}

func personalChannel(cfg *config.Config) string {
	// While this is not truly "personal" channel as there will be
	// channel name collisions among maxBenchmarkClients this is ok
	// for our use case where we really just want a distribution close
	// to one unique channel per client and a way to publish messages
	// into personal channels.
	return "personal" + strconv.Itoa(rand.Intn(numClientConnections*cfg.Replicas*2))
}

func groupChannel() string {
	// Each client will be subscribed to group channel. The amount of
	// subscribers in such group will be close to replica number of
	// client pods. For example if we have 100 clients pods to generate
	// maxBenchmarkClients connections then every group channel will
	// contain about 100 subscribers.
	return "group" + strconv.Itoa(rand.Intn(numClientConnections*2))
}

func runSubscriber(cfg *config.Config, url string, num int) *centrifuge.Client {
	client := centrifuge.New(url, centrifuge.DefaultConfig())

	personalSub, _ := client.NewSubscription(personalChannel(cfg))
	personalSub.OnPublish(&eventHandler{"personal"})
	personalSub.Subscribe()

	groupSub, _ := client.NewSubscription(groupChannel())
	groupSub.OnPublish(&eventHandler{"group"})
	groupSub.Subscribe()

	client.Connect()

	go func() {
		// Periodically disconnect and connect back.
		for {
			min := 10
			max := 10 * 60
			ttlSeconds := rand.Intn(max-min) + min
			time.Sleep(time.Duration(ttlSeconds) * time.Second)
			client.Disconnect()
			time.Sleep(time.Second)
			client.Connect()
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

func runPersonalPublisher(cfg *config.Config, url string, sleep time.Duration) *centrifuge.Client {
	client := centrifuge.New(url, centrifuge.DefaultConfig())
	client.Connect()

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
			err := client.Publish(personalChannel(cfg), data)
			if err != nil {
				statsdClient.Increment("publish.personal_channel.error")
			}
		}
	}()
	return client
}

func runGroupPublisher(cfg *config.Config, url string, sleep time.Duration) *centrifuge.Client {
	client := centrifuge.New(url, centrifuge.DefaultConfig())
	client.Connect()

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
			err := client.Publish(groupChannel(), data)
			if err != nil {
				statsdClient.Increment("publish.group_channel.error")
			}
		}
	}()
	return client
}

// NewStatsd возвращает новый настроенный клиент statsd
func NewStatsd(host string, port int, protocol string, prefix string, enable bool) (*statsd.Client, error) {
	protocol = strings.ToLower(protocol)

	client, err := statsd.New(
		statsd.Address(fmt.Sprintf("%v:%v", host, port)),
		statsd.Prefix(prefix),
		statsd.Network(protocol),
		statsd.Mute(!enable))

	if err != nil {
		return nil, err
	}

	return client, nil
}

func waitExitSignal(clients []*centrifuge.Client) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		go func() {
			time.Sleep(20 * time.Second)
			os.Exit(1)
		}()
		for _, c := range clients {
			c.Close()
		}
		done <- true
	}()
	<-done
}

func prepareForGraphite(s string) string {
	if s == "" {
		return "_"
	}
	return strings.Replace(s, ".", "_", -1)
}
