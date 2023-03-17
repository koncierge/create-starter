package main

import (
	"math"
	"os"
	"time"
)

/* Data Handling */

func i32tob(val uint32) byte {
	if val > 65535 {
		return 255
	}

	return byte(val >> 8)
}

func byteOrder(i uint16) uint16 {
	return (i<<8)&0xff00 | i>>8
}

/* Time & Date */

func today() string {
	now := time.Now()
	return now.Format("2006-01-02")
}

/* File System */

func fileExists(path string) bool {
	_, error := os.Stat(path)

	// check if error is "file not exists"
	if os.IsNotExist(error) {
		return false
	} else {
		return true
	}
}

/* Colour & Light */

func gamma(val float64) float64 {
	depth := float64(65535)
	compressed := math.Pow((val/depth), (1/2.8)) * depth // Assuming gamma value of 2.8 for an LED displays

	return math.Round(compressed)
}

func degamma(val float64) float64 {
	depth := float64(65535)
	expanded := math.Pow((val/depth), 2.2) * depth // Assuming gamma value of 2.2 for RGB encoding

	return math.Round(expanded)
}

func raw(val uint32) uint32 {
	luma := float64(val)
	raw := degamma(luma)

	return uint32(math.Round(float64(raw)))
}

func power(val uint32, multiplier float64) uint32 {
	luma := float64(val)
	pow := gamma(dim(luma, multiplier))

	return uint32(math.Round(float64(pow)))
}

func dim(val float64, multiplier float64) float64 {
	// dimmed := ((val / 100) * float64(brightness.get())) * multiplier

	// Brightness now controlled by brightness packet
	return val * multiplier
}
