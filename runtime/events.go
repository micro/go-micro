package runtime

const (
	// CreatedEvent is published when a service is created
	CreatedEvent string = "runtime.service.created"
	// UpdatedEvent is published when a service is updated
	UpdatedEvent = "runtime.service.updated"
	// DeletedEvent is published when a service is updated
	DeletedEvent = "runtime.service.deleted"
)

// EventPayload contains the data which is published with an event.
type EventPayload struct {
	// Service which the event relates to
	Service *Service
	// Options used to create the service
	Options *CreateOptions
}
