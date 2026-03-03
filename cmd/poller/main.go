package main

import (
	"context"

	packagewatcher "git.duti.dev/secure-package-registry/pkg/services/package-watcher"
)

func main() {
	packagewatcher, err := packagewatcher.NewWatcher()
	if err != nil {
		panic(err)
	}

	if err := packagewatcher.Start(context.Background()); err != nil {
		panic(err)
	}
}
