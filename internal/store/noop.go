package store

import "context"

type Noop struct{}

func (Noop) RecordEvent(context.Context, Event) error                 { return nil }
func (Noop) UpsertSession(context.Context, SessionRecord) error       { return nil }
func (Noop) History(context.Context, string) ([]HistoryRecord, error) { return nil, nil }
func (Noop) SessionDetails(context.Context, string) (SessionDetails, error) {
	return SessionDetails{}, nil
}
func (Noop) TopCommands(context.Context, string, int) ([]CommandStat, error) {
	return nil, nil
}
func (Noop) Stats(context.Context, string) (Stats, error) { return Stats{}, nil }
func (Noop) Purge(context.Context, string) error          { return nil }
func (Noop) Close() error                                 { return nil }
