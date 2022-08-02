package messagebroker

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Export as env variable in container/ pod:
var uname = os.Getenv("RABBITMQ_USERNAME")
var psswd = os.Getenv("RABBITMQ_PASSWORD")
var endpoint = os.Getenv("RABBITMQ_ENDPOINT")

var AMQP_SERVER_URL = fmt.Sprintf("amqp://%s:%s@%s", uname, psswd, endpoint)

type Message struct {
	ExchangeName string
	Message      map[string]interface{}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func ConnectToRabbitMQ(exchangeName string) (*amqp.Channel, error) {
	// Create a new RabbitMQ connection.
	connectRabbitMQ, err := amqp.Dial(AMQP_SERVER_URL)
	if err != nil {
		return nil, err
	}

	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		return nil, err
	}
	// defer channelRabbitMQ.Close()

	// With the instance and declare Queues that we can
	// publish and subscribe to.)
	err = channelRabbitMQ.ExchangeDeclare(
		exchangeName,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	return channelRabbitMQ, err

}

func PublishMessage(message Message) error {
	ch, err := ConnectToRabbitMQ(message.ExchangeName)
	defer ch.Close()
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ due to %v", err.Error())
		return err
	}
	jsonStr, _ := json.Marshal(message.Message)
	fmt.Printf("Publishing message: %v", message.Message)
	err = ch.Publish(
		message.ExchangeName, // exchange
		"",                   // routing key
		false,                // mandatory
		false,                // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			AppId:        "remote-build",
			ContentType:  "application/json",
			Body:         []byte(jsonStr),
		})
	if err != nil {
		fmt.Printf("Failed to publish: %v", err.Error())
	}

	return err
}
