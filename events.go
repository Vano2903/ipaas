package main

import (
	"github.com/docker/docker/api/types"
	"log"
)

func (c ContainerController) EventHandler() {
	eventChan, errChan := c.cli.Events(c.ctx, types.EventsOptions{})
	for {
		select {
		case event := <-eventChan:
			//log.Println(event)
			switch event.Type {
			case "container":
				//log.Println("container event\n\n")
				switch event.Action {
				case "die":
					log.Println("[EVENT] Container died:", event.Actor.ID)
				case "health_status":
					log.Println("[EVENT] Container health status:", event.Actor.ID)
				case "kill":
					log.Println("[EVENT] Container killed:", event.Actor.ID)
				case "update":
					log.Println("[EVENT] Container updated:", event.Actor.ID)
				}
			}
		case err := <-errChan:
			log.Println("[errore] Error in event handler: ", err)
		}
	}
}
