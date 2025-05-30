package event

const (
	// CloudEvent type templates
	// Real‐Twin events:      twinscale.real.<twin-interface>
	EVENT_TYPE_REAL_GENERATED = "twinscale.real.%s"
	// Event-Store events:    twinscale.eventstore.<twin-interface>
	EVENT_TYPE_EVENTSTORE_GENERATED = "twinscale.eventstore.%s"
	// Historical-Store events: twinscale.historicalstore.<twin-interface>
	EVENT_TYPE_HISTORICAL_GENERATED = "twinscale.historicalstore.%s"
	// Notification events:   twinscale.notification.<twin-interface>
	EVENT_TYPE_NOTIFICATION = "twinscale.notification.%s"
	// Visualisation events:  twinscale.visualisation.<twin-interface>
	EVENT_TYPE_VISUALISATION = "twinscale.visualisation.%s"
	// Command events:        twinscale.command.<twin-interface>.<command>
	EVENT_TYPE_COMMAND_EXECUTED = "twinscale.command.%s.%s"
)

const (
	// Dispatcher names & queues
	CLOUD_EVENT_DISPATCHER       = "cloud-event-dispatcher"
	MQTT_DISPATCHER              = "mqtt-dispatcher"
	CLOUD_EVENT_DISPATCHER_QUEUE = "cloud-event-dispatcher-queue"
	MQTT_DISPATCHER_QUEUE        = "mqtt-dispatcher-queue"
)

const (
	// Broker & RabbitMQ defaults
	EVENT_BROKER_NAME               = "twinscale"
	RABBITMQ_VHOST                  = "/"
	CLOUD_EVENT_DISPATCHER_EXCHANGE = "amq.topic"
	MQTT_EXCHANGE                   = "amq.topic"
)
