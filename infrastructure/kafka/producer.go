package kafka

import (
	"fmt"

	"github.com/Shopify/sarama"
)

type Producer interface {
	SendMessage(topic string, message string) error
}

type producer struct {
	producer sarama.SyncProducer
}

func NewProducer(kafkaBrokers []string) (Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	kafkaProducer, err := sarama.NewSyncProducer(kafkaBrokers, config)

	return producer{kafkaProducer}, err
}

func (p producer) SendMessage(topic string, message string) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
	}

	p, o, err := p.producer.SendMessage(msg)
	if err != nil {
		fmt.Println("Error publish: ", err.Error())
	}

	fmt.Println("Partition: ", p)
	fmt.Println("Offset: ", o)
}
