package dockwatch

import (
	"context"
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
	Original  types.Container // only set for UPDATE events
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

	for _, n := range next {
		exists := false
		var c types.Container
		for _, c = range current {
			if n.ID == c.ID {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, Event{
				Type:      EventTypeCreate,
				Container: n,
			})
		} else {
			if !containersEqual(n, c) {
				result = append(result, Event{
					Type:      EventTypeUpdate,
					Container: c,
					Original:  n,
				})
			}
		}
	}

	for _, c := range current {
		exists := false
		for _, n := range next {
			if c.ID == n.ID {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, Event{
				Type:      EventTypeDelete,
				Container: c,
			})
		}
	}

	return
}

func containersEqual(a, b types.Container) bool {
	// easy ones first
	if a.ID != b.ID ||
		a.Image != b.Image ||
		a.ImageID != b.ImageID ||
		a.Command != b.Command ||
		a.State != b.State ||
		a.Status != b.Status ||
		a.SizeRw != b.SizeRw ||
		a.SizeRootFs != b.SizeRootFs {
		return false
	}

	aPorts := ports(a.Ports)
	bPorts := ports(b.Ports)
	sort.Sort(aPorts)
	sort.Sort(bPorts)

	aMounts := mounts(a.Mounts)
	bMounts := mounts(b.Mounts)
	sort.Sort(aMounts)
	sort.Sort(bMounts)

	return reflect.DeepEqual(a.Names, b.Names) &&
		reflect.DeepEqual(a.Created, b.Created) &&
		reflect.DeepEqual(aPorts, bPorts) &&
		reflect.DeepEqual(a.Labels, b.Labels) &&
		reflect.DeepEqual(a.HostConfig, b.HostConfig) &&
		reflect.DeepEqual(a.NetworkSettings, b.NetworkSettings) &&
		reflect.DeepEqual(aMounts, bMounts)
}

// Because the Docker API seems to randomly change the order of the `ports`
// field, and arrays are sorted, this results in random "update" events. To
// solve this, the above function has to compare each field individually and the
// array fields must be sorted, hence the custom type and sort impl.
type ports []types.Port

func (l ports) Len() int           { return len(l) }
func (l ports) Less(i, j int) bool { return l[i].PrivatePort < l[j].PrivatePort }
func (l ports) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

type mounts []types.MountPoint

func (l mounts) Len() int           { return len(l) }
func (l mounts) Less(i, j int) bool { return l[i].Destination < l[j].Destination }
func (l mounts) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
