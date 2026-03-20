package state

// EventStore is the append-only event log (source of truth).
// All state changes flow through this interface as immutable events.
type EventStore interface {
	Append(evt Event) error
	List(filter EventFilter) ([]Event, error)
	Count(filter EventFilter) (int, error)
	All() ([]Event, error)
}

// ProjectionStore provides queryable materialized views of domain state.
// Events are projected into denormalized tables for efficient querying.
type ProjectionStore interface {
	Project(evt Event) error
	GetRequirement(id string) (Requirement, error)
	GetStory(id string) (Story, error)
	ListRequirements(filter ReqFilter) ([]Requirement, error)
	ListStories(filter StoryFilter) ([]Story, error)
	ListAgents(filter AgentFilter) ([]Agent, error)
	ListEscalations() ([]Escalation, error)
	ListStoryDeps(reqID string) ([]StoryDep, error)
	ArchiveRequirement(reqID string) error
	ArchiveStoriesByReq(reqID string) error
	Close() error
}
