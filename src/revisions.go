package main

import "sync"

var (
	revisionMu        sync.Mutex
	settingsRevision  int64 = 1
	providersRevision int64 = 1
)

func currentSettingsRevision() int64 {
	revisionMu.Lock()
	defer revisionMu.Unlock()
	return settingsRevision
}

func currentProvidersRevision() int64 {
	revisionMu.Lock()
	defer revisionMu.Unlock()
	return providersRevision
}

func bumpSettingsRevision() int64 {
	revisionMu.Lock()
	defer revisionMu.Unlock()
	settingsRevision++
	return settingsRevision
}

func bumpProvidersRevision() int64 {
	revisionMu.Lock()
	defer revisionMu.Unlock()
	providersRevision++
	return providersRevision
}
