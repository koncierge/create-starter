package main

import (
	"log"
	"os"
	"path/filepath"
	"reflect"
)

/* Manifest Register */

type register struct {
	files []string
	data  [][][]uint32
}

/* Methods */

func (r *register) init() {
	files := listFiles()

	r.files = files

	log.Println("[INFO] Manifest initiated with:")
	log.Println(r.files)

	r.decode()
}

func (r *register) update() bool {
	latest := listFiles()

	if !reflect.DeepEqual(r.files, latest) {
		r.files = latest

		log.Println("[INFO] File changes detected, manifest updated:")
		log.Println(latest)

		return true
	}

	return false
}

func (r *register) decode() {
	decode(r)

	if len(r.data) > 0 || usingSystemFallback(r.files) {
		return
	}

	log.Println("[ERROR] No streamable assets decoded, loading fallback")
	r.files = fallbackFiles()
	decode(r)
}

/* Helpers */

func listFiles() []string {
	entries, err := os.ReadDir(assets)

	var list []string

	if err != nil {
		log.Println("[ERROR] Unable to access asset path, using fallback")
		list = fallbackFiles()
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			list = append(list, filepath.Join(assets, entry.Name()))
		}
	}

	if len(list) == 0 {
		log.Println("[INFO] No assets found, using default slide")
		list = defaultFiles()
	}

	return list
}

func defaultFiles() []string {
	if defaultSlide != "" {
		return []string{defaultSlide}
	}

	log.Println("[INFO] DEFAULT_SLIDE is not configured, using fallback")
	return fallbackFiles()
}

func fallbackFiles() []string {
	return []string{systemFallback}
}

func usingSystemFallback(files []string) bool {
	return reflect.DeepEqual(files, fallbackFiles())
}
