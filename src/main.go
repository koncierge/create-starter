package main

import (
	"time"
)

func main() {

	/* Environment */
	setup()
	defer logFile.Close()

	// Create image manifest
	manifest.init()

	/* Execute */
	loop()
}

func loop() {

	for {

		tick()

		if len(manifest.data) == 0 {
			time.Sleep(time.Duration(duration) * time.Second)
			continue
		}

		// Stream images to display
		for _, img := range manifest.data {
			streamImg(img, duration)
		}

	}
}

/* Helpers */

func tick() {

	// Check time and update brightness
	daytime.update()
	brightness.update()

	// Check for new slides
	updated := manifest.update()

	// Decode updated images
	if updated {
		manifest.decode()
	}
}
