package messanger

import "github.com/streadway/amqp"

type Messanger struct {
	AmpqUri            string
	ListeningQueueUri  string
	RespondingQueueUri string
	ListeningQueue     *amqp.Queue
	RespondingQueue    *amqp.Queue
	conn               *amqp.Connection
	channel            *amqp.Channel
}

func NewMessanger(ampqUri string, listeningQueueUri string, respondingQueueUri string) *Messanger {
	return &Messanger{
		AmpqUri:            ampqUri,
		ListeningQueueUri:  listeningQueueUri,
		RespondingQueueUri: respondingQueueUri,
	}
}

func (m *Messanger) Connect() error {
	var err error
	m.conn, err = amqp.Dial(m.AmpqUri)
	if err != nil {
		return err
	}
	m.channel, err = m.conn.Channel()
	if err != nil {
		return err
	}
	return nil
}

func (m *Messanger) Close() error {
	err := m.channel.Close()
	if err != nil {
		return err
	}
	err = m.conn.Close()
	if err != nil {
		return err
	}
	return nil
}

func (m *Messanger) DeclareListeningQueue() error {
	q, err := m.channel.QueueDeclare(
		m.ListeningQueueUri, // name
		false,               // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return err
	}
	m.ListeningQueue = &q
	return nil
}

func (m *Messanger) DeclareRespondingQueue() error {
	q, err := m.channel.QueueDeclare(
		m.RespondingQueueUri, // name
		false,                // durable
		false,                // delete when unused
		false,                // exclusive
		false,                // no-wait
		nil,                  // arguments
	)
	if err != nil {
		return err
	}
	m.RespondingQueue = &q
	return nil
}

func (m *Messanger) Listen() (<-chan amqp.Delivery, error) {
	msgs, err := m.channel.Consume(
		m.ListeningQueue.Name, // queue
		"",                    // consumer
		true,                  // auto-ack
		false,                 // exclusive
		false,                 // no-local
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

func (m *Messanger) Respond(msg string) error {
	err := m.channel.Publish(
		"",                     // exchange
		m.RespondingQueue.Name, // routing key
		false,                  // mandatory
		false,                  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(msg),
		})
	if err != nil {
		return err
	}
	return nil
}
