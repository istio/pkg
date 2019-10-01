package ledger

import (
	"fmt"
	"istio.io/istio/galley/pkg/runtime/resource"
)

type ConfigTracker interface {
	Handle(e resource.Event)
	VersionContainsConfig(configHash string, resourceName string, resourceVersion string) bool
}

type ConfigTrackerDefault struct {
	ledger Ledger
}

func (c *ConfigTrackerDefault) Handle(e resource.Event) {
	switch e.Kind {
	case resource.Added, resource.Updated:
		_, err := c.ledger.Put(e.Entry.ID.FullName.String(), string(e.Entry.ID.Version))
		logIfError(err, e)
	case resource.Deleted, resource.FullSync:
		err := c.ledger.Delete(e.Entry.ID.FullName.String())
		logIfError(err, e)
	default:
		// ignore other events
	}
}

func (c *ConfigTrackerDefault) VersionContainsConfig(configHash string, resourceName string, resourceVersion string) bool {
	res, err := c.ledger.GetPreviousValue(configHash, resourceName)
	if err != nil {
		// TODO: what is the right thing here?
	}
	return res == resourceVersion
}

func logIfError(e error, event resource.Event) {
	// TODO: Make this real logging
	if e!=nil{
		fmt.Printf("The Config History Cache encountered an error while handling event (%v): %s", event, e)
	}
}