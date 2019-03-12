package server

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"spaceship/model"
)

type PubSub struct {
	pubChan *amqp.Channel
	subChan *amqp.Channel
	sessionHolder *SessionHolder
	logger *Logger
}

func NewPubSub(sessionHolder *SessionHolder, logger *Logger) *PubSub {

	conn, err := amqp.Dial("amqp://qqbrajzx:wDiFe2gSdDxbjkFyb3mr_4raCux9VXh7@bear.rmq.cloudamqp.com/qqbrajzx")
	if err != nil {
		logger.Fatalw("Error while trying to connect amqp server", "error", err)
	}
	//TODO: don't forget to close this connection

	pubChan, err := conn.Channel()
	if err != nil {
		logger.Fatalw("Error while trying to open a channel for publish over amqp connection", "error", err)
	}

	subChan, err := conn.Channel()
	if err != nil {
		logger.Fatalw("Error while trying to open a channel for subscibe over amqp connection", "error", err)
	}

	//Now we should define exchange for both channels
	err = pubChan.ExchangeDeclare(
			"messages",
			"fanout",
			true,
			false,
			false,
			false,
			nil,
		)
	if err != nil {
		logger.Fatalw("Error while trying to define exchange over publish channel", "error", err)
	}

	err = subChan.ExchangeDeclare(
		"messages",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Fatalw("Error while trying to define exchange over subscribe channel", "error", err)
	}


	q, err := subChan.QueueDeclare(
			"",
			false,
			false,
			true,
			false,
			nil,
		)
	if err != nil {
		logger.Fatalw("Error while trying to define queue over subscribe channel", "error", err)
	}

	err = subChan.QueueBind(
			q.Name,
			"",
			"messages",
			false,
			nil,
		)
	if err != nil {
		logger.Fatalw("Error while binding queue to subscribe channel", "error", err)
	}

	msgs, err := subChan.Consume(
			q.Name,
			"",
			true,
			false,
			false,
			false,
			nil,
		)
	if err != nil{
		logger.Fatalw("Error while trying to create consumer channel on subscribe channel", "error", err)
	}

	go func(){

		defer conn.Close()

		for msg := range msgs {

			if msg.ContentType == "application/json" {

				msgModel := model.PubSubMessage{}

				err := json.Unmarshal(msg.Body, &msgModel)
				if err != nil {
					logger.Errorw("Error while unmarshal pub sub message data", "error", err)
					continue
				}

				for _, userID := range msgModel.UserIDs {

					session := sessionHolder.GetByUserID(userID)
					if session != nil {
						msgData, err := json.Marshal(msgModel.Data)
						if err != nil {
							logger.Errorw("Error while trying to marshal message data", "error", err)
							break
						}
						_ = session.SendBytes(false, 0, msgData)
					}

				}

			}else{
				logger.Errorw("Unrecognized content type received", "content-type", msg.ContentType)
			}

		}

	}()


	return &PubSub{
		sessionHolder: sessionHolder,
		logger: logger,
		pubChan: pubChan,
		subChan: subChan,
	}

}

func (ps *PubSub) Send(message *model.PubSubMessage) error {

	data, err := json.Marshal(message)
	if err != nil {
		ps.logger.Errorw("Error while trying to marshal message in send method of pubsub module", "error", err)
		return err
	}

	err = ps.pubChan.Publish(
		"messages",
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body: data,
		})

	if err != nil {
		ps.logger.Errorw("Error while trying to publish data in send method of pubsub module", "error", err)
		return err
	}

	return nil

}

