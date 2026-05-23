package vo

type SagaState string

const (
	SagaStateStarted          SagaState = "STARTED"
	SagaStateWaitingForEscrow SagaState = "WAITING_FOR_ESCROW"
	SagaStateWaitingForChat   SagaState = "WAITING_FOR_CHAT"
	SagaStateCompleted        SagaState = "COMPLETED"
	SagaStateRevertingEscrow  SagaState = "REVERTING_ESCROW"
	SagaStateFailedTechnical  SagaState = "FAILED_TECHNICAL"
	SagaStateFailedRequiresAdmin SagaState = "FAILED_REQUIRES_ADMIN"
)
