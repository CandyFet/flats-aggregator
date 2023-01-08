package kafka

import (
	"fmt"
	"log"

	"github.com/Shopify/sarama"
)

type Producer interface {
	SendMessage(topic, message, partitionKey string) error
}

type producer struct {
	producer sarama.SyncProducer
	logger   *log.Logger
}

func NewProducer(kafkaBrokers []string, logger *log.Logger) (Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	kafkaProducer, err := sarama.NewSyncProducer(kafkaBrokers, config)

	return producer{kafkaProducer, logger}, err
}

func (p producer) SendMessage(topic, message, partitionKey string) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
		Key:   sarama.StringEncoder(partitionKey),
	}

	prt, ofs, err := p.producer.SendMessage(msg)

	if err != nil {
		fmt.Println("Error publish: ", err.Error())
		return err
	}

	p.logger.Printf("message successfully published to topic %s, in partition %d, with offset %d", topic, prt, ofs)

	return nil
}
