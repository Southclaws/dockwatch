package dockerwatch

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
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

type label struct {
	container types.Container
	key       string
	value     string
}

type labels []label

func (l labels) Len() int           { return len(l) }
func (l labels) Less(i, j int) bool { return l[i].container.ID < l[j].container.ID }
func (l labels) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

// Watcher represents a daemon that watches containers for a specific label
type Watcher struct {
	Events chan Event // Events is where container change events are published
	Errors chan error

	docker  *client.Client
	labels  []string
	current []label // current label value for each container
}

// New starts a new watcher which calls fn when
func New(docker *client.Client, labels []string) (w *Watcher) {
	w = &Watcher{
		Events: make(chan Event, 16),
		Errors: make(chan error, 16),
		docker: docker,
		labels: labels,
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
	containers, err := w.docker.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return
	}

	var next []label
	for _, container := range containers {
		for k, v := range container.Labels {
			next = append(next, label{container: container, key: k, value: v})
		}
	}

	for _, change := range diff(w.current, next) {
		fmt.Println(change)
	}

	return
}

func diff(current labels, next labels) (result []Event) {
	if len(current) == 0 && len(next) == 0 {
		return
	}

	sort.Sort(current)
	sort.Sort(next)

	for _, o := range current {
		exists := false
		var t label
		for _, t = range next {
			if o.container.ID == t.container.ID {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, Event{
				Type:      EventTypeCreate,
				Container: o.container,
			})
		} else {
			if !reflect.DeepEqual(o, t) {
				result = append(result, Event{
					Type:      EventTypeUpdate,
					Container: o.container,
				})
			}
		}
	}

	for _, t := range next {
		exists := false
		for _, o := range current {
			if t.container.ID == o.container.ID {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, Event{
				Type:      EventTypeDelete,
				Container: t.container,
			})
		}
	}

	return
}
