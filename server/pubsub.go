package server

import (
	"bytes"
	"context"
	"github.com/golang/protobuf/jsonpb"
	"github.com/streadway/amqp"
	"spaceship/socketapi"
)

type PubSub struct {
	isEnabled bool
	pubChan *amqp.Channel
	subChan *amqp.Channel
	sessionHolder *SessionHolder
	logger *Logger
	jsonpbMarshler *jsonpb.Marshaler
	jsonpbUnmarshaler *jsonpb.Unmarshaler
	context context.Context
}

func NewPubSub(config *Config, sessionHolder *SessionHolder, jsonpbMarshler *jsonpb.Marshaler, jsonpbUnmarshaler *jsonpb.Unmarshaler, logger *Logger, context context.Context) *PubSub {

	if config.RabbitMQ.ConnectionString != "" {
		conn, err := amqp.Dial(config.RabbitMQ.ConnectionString)
		if err != nil {
			logger.Fatalw("Error while trying to connect amqp server", "error", err)
		}

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

			for {

				select {
				case <- context.Done():
					logger.Info("Exiting from subscribe routine")
					return
				case msg := <- msgs:

					if msg.ContentType == "application/json" {

						msgModel := &socketapi.PubSubMessage{}

						err := jsonpbUnmarshaler.Unmarshal(bytes.NewReader(msg.Body), msgModel)
						if err != nil {
							logger.Errorw("Error while unmarshal pub sub message data", "error", err)
							continue
						}

						for _, userID := range msgModel.UserIDs {

							session := sessionHolder.GetByUserID(userID)
							if session != nil {
								_ = session.Send(false, 0, msgModel.Data)
							}

						}

					}else{
						logger.Errorw("Unrecognized content type received", "content-type", msg.ContentType)
					}
				}

			}

		}()


		return &PubSub{
			isEnabled: true,
			sessionHolder: sessionHolder,
			logger: logger,
			pubChan: pubChan,
			subChan: subChan,
			jsonpbMarshler: jsonpbMarshler,
			jsonpbUnmarshaler: jsonpbUnmarshaler,
			context: context,
		}
	}else{
		return &PubSub{
			isEnabled: false,
			sessionHolder: sessionHolder,
			logger: logger,
			jsonpbMarshler: jsonpbMarshler,
			jsonpbUnmarshaler: jsonpbUnmarshaler,
			context: context,
		}
	}

}

func (ps *PubSub) Send(message *socketapi.PubSubMessage) error {

	//We can check the user ids to select the ones already belongs to current node
	//So, we can send message to them directly instead of publishing

	publishUserIDs := make([]string, 0)

	for _, userID := range message.UserIDs {

		session := ps.sessionHolder.GetByUserID(userID)
		if session != nil {
			_ = session.Send(false, 0, message.Data)
		}else{
			publishUserIDs = append(publishUserIDs, userID)
		}

	}

	//If we still have user IDs remaining and if pubsub module is enabled we need to publish them
	if ps.isEnabled && len(publishUserIDs) > 0 {

		message.UserIDs = publishUserIDs
		data, err := ps.jsonpbMarshler.MarshalToString(message)
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
				Body: []byte(data),
			})

		if err != nil {
			ps.logger.Errorw("Error while trying to publish data in send method of pubsub module", "error", err)
			return err
		}
	}

	return nil

}

