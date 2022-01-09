# Kafka Broker


## Async Publish
```go
import "github.com/Shopify/sarama"

func AsyncProduceMessage()  {
    var errorsChan = make(chan *sarama.ProducerError)
    var successesChan = make(chan *sarama.ProducerMessage)
    go func() {
        for err := range errorsChan {
            fmt.Println(err)
        }
    }
    go func() {
        for v := range successesChan {
            fmt.Println(v)
        }
    }
    b := NewBroker(AsyncProducer(errorsChan,successesChan))
    b.Publish(`topic`, &broker.Message{})
}
```
