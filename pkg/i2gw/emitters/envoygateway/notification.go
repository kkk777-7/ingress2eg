package envoygateway_emitter

import (
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func notify(mType notifications.MessageType, message string, callingObject ...client.Object) {
	newNotification := notifications.NewNotification(mType, message, callingObject...)
	notifications.NotificationAggr.DispatchNotification(newNotification, emitterName)
}
