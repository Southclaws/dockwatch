package main

import (
	"log"

	"github.com/Southclaws/dockwatch"
	"github.com/docker/docker/client"
	"github.com/kr/pretty"
)

func main() {
	c, err := client.NewEnvClient()
	if err != nil {
		log.Fatal(err)
	}

	w := dockwatch.New(c)

	for {
		select {
		case e := <-w.Events:
			if e.Type == dockwatch.EventTypeUpdate {
				log.Printf("%s: %v (%s)\n%s\n", e.Type, e.Container.Names, e.Container.ID, pretty.Sprint(pretty.Diff(e.Container, e.Original)))
			} else {
				log.Printf("%s: %v (%s)\n%s\n", e.Type, e.Container.Names, e.Container.ID, pretty.Sprint(e.Container))
			}
		case e := <-w.Errors:
			log.Println("Error:", e)
		}
	}
}
