package main

import (
	"log"

	"github.com/Southclaws/dockwatch"
	"github.com/docker/docker/client"
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
			log.Printf("%s: %v (%s)\n", e.Type, e.Container.Names, e.Container.ID)
		case e := <-w.Errors:
			log.Println("Error:", e)
		}
	}
}
