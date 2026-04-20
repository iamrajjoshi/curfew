package store

import "context"

type Noop struct{}

func (Noop) RecordEvent(context.Context, Event) error              { return nil }
func (Noop) UpsertSession(context.Context, SessionRecord) error    { return nil }
func (Noop) History(context.Context, int) ([]HistoryRecord, error) { return nil, nil }
func (Noop) Stats(context.Context, int) (Stats, error)             { return Stats{}, nil }
func (Noop) Purge(context.Context, int) error                      { return nil }
func (Noop) Close() error                                          { return nil }
