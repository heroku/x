package saramahelper

import (
	"github.com/Shopify/sarama"
	"github.com/heroku/x/hkafka"
	"github.com/pkg/errors"
)

// NewAsyncProducer from the provided heroku kafka and sarama configs The sarama config's TLS.Config and TLS.Enabled are
// changed, but all other configuration options (like ClientID, Producer.Return.Errors, Producer.RequiredAcks, etc) must
// be set by the caller.
func NewAsyncProducer(hkc hkafka.Config, sc *sarama.Config) (sarama.AsyncProducer, error) {
	if err := hkc.VerifyServers(); err != nil {
		return nil, errors.Wrap(err, "verifying servers")
	}

	c, err := hkc.TLSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "creating tls config")
	}
	sc.Net.TLS.Config = c
	sc.Net.TLS.Enable = true

	b, err := hkc.BrokerAddresses()
	if err != nil {
		return nil, errors.Wrap(err, "determining broker addresses")
	}

	return sarama.NewAsyncProducer(b, sc)
}

func NewAsyncProducerFromDefaultConfig(consumerGroup string) (sarama.AsyncProducer, error) {
	hc, err := hkafka.NewConfigFromEnv()
	if err != nil {
		return nil, errors.Wrap(err, "generating config from env")
	}

	sc := sarama.NewConfig()
	sc.Producer.Return.Errors = true
	sc.Producer.RequiredAcks = sarama.WaitForAll // Default is WaitForLocal
	sc.ClientID = consumerGroup

	return NewAsyncProducer(hc, sc)
}
