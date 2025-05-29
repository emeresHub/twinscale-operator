package naming

import (
	"fmt"

	"github.com/emereshub/twinscale-operator/pkg/event"
)

func GetEventTypeRealGenerated(twinInterfaceName string) string {
	return fmt.Sprintf(event.EVENT_TYPE_REAL_GENERATED, twinInterfaceName)
}

func GetEventTypeEventStoreGenerated(twinInterfaceName string) string {
	return fmt.Sprintf(event.EVENT_TYPE_EVENTSTORE_GENERATED, twinInterfaceName)
}

func GetEventTypeHistoricalGenerated(twinInterfaceName string) string {
	return fmt.Sprintf(event.EVENT_TYPE_HISTORICAL_GENERATED, twinInterfaceName)
}

func GetEventTypeNotification(twinInterfaceName string) string {
	return fmt.Sprintf(event.EVENT_TYPE_NOTIFICATION, twinInterfaceName)
}

func GetEventTypeVisualisation(twinInterfaceName string) string {
	return fmt.Sprintf(event.EVENT_TYPE_VISUALISATION, twinInterfaceName)
}

// Aliases for MQTT dispatcher use
func GetNewCloudEventEventBinding(twinInterfaceName string) string {
	return GetEventTypeRealGenerated(twinInterfaceName)
}

func GetNewMQQTEventBinding(twinInterfaceName string) string {
	return GetEventTypeRealGenerated(twinInterfaceName)
}
