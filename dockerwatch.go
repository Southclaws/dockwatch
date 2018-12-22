package dockwatch

import (
	"context"
	"reflect"
	"sort"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/kr/pretty"
)

// EventType represents types of change events
type EventType string

// Event types
const (
	EventTypeCreate EventType = "CREATE"
	EventTypeUpdate EventType = "UPDATE"
	EventTypeDelete EventType = "DELETE"
)

// Event represents a container label change
type Event struct {
	Type      EventType
	Container types.Container
}

type containers []types.Container

func (l containers) Len() int           { return len(l) }
func (l containers) Less(i, j int) bool { return l[i].ID < l[j].ID }
func (l containers) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

// Watcher represents a daemon that watches containers for a specific label
type Watcher struct {
	Events chan Event // Events is where container change events are published
	Errors chan error

	docker  *client.Client
	current []types.Container
}

// New starts a new watcher which calls fn when
func New(docker *client.Client) (w *Watcher) {
	w = &Watcher{
		Events: make(chan Event, 16),
		Errors: make(chan error, 16),
		docker: docker,
	}
	go w.start()
	return
}

func (w *Watcher) start() {
	for range time.NewTicker(time.Second).C {
		err := w.check()
		if err != nil {
			w.Errors <- err
		}
	}
	return
}

func (w *Watcher) check() (err error) {
	next, err := w.docker.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return
	}

	for _, change := range diff(w.current, next) {
		w.Events <- change
	}

	w.current = next

	return
}

func diff(current containers, next containers) (result []Event) {
	if len(current) == 0 && len(next) == 0 {
		return
	}

	sort.Sort(current)
	sort.Sort(next)

	for _, c := range next {
		exists := false
		var n types.Container
		for _, n = range current {
			if c.ID == n.ID {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, Event{
				Type:      EventTypeCreate,
				Container: c,
			})
		} else {
			if !reflect.DeepEqual(c, n) {
				result = append(result, Event{
					Type:      EventTypeUpdate,
					Container: c,
				})

				pretty.Println(pretty.Diff(c, n))
			}
		}
	}

	for _, n := range current {
		exists := false
		for _, c := range next {
			if n.ID == c.ID {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, Event{
				Type:      EventTypeDelete,
				Container: n,
			})
		}
	}

	return
}
