package main

import (
	"image"
	_ "image/gif" // Import but don't use
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"

	"github.com/disintegration/imaging"
)

func decode(manifest *register) {
	manifest.data = [][][]uint32{}

	for _, path := range manifest.files {

		// Open the file
		file, err := os.Open(path)
		if err != nil {
			log.Println("[ERROR] Unable to open image:", err)
			continue
		}

		// Check file format
		if checkFmt(file) {

			// Load the image
			img, _, err := image.Decode(file)
			if err != nil {
				log.Println("[ERROR] Unable to decode image:", err)
				file.Close()
				continue
			}

			bounds := img.Bounds()

			// Check size, adjust if required
			if !checkSize(bounds) {
				img = resizeImg(img, bounds)
			}

			// Decode pixel data
			var imgData [][]uint32

			for y := 0; y < img.Bounds().Dy(); y++ {
				var row []uint32

				for x := 0; x < img.Bounds().Dx(); x++ {
					pixel := img.At(x, y)
					r, g, b, _ := pixel.RGBA()
					row = append(row, raw(b), raw(g), raw(r)) // Note: RGBA() returns colors in the range [0, 65535], needs conversion to [0, 255] + gamma correction
				}

				imgData = append(imgData, row)
			}

			manifest.data = append(manifest.data, imgData)

		}

		err = file.Close()
		if err != nil {
			log.Println("[ERROR] Unable to close image:", err)
		}
	}
}

/* Helpers */

func checkFmt(file *os.File) bool {
	_, format, err := image.DecodeConfig(file)
	if err != nil {
		file.Seek(0, 0)
		return false
	}

	file.Seek(0, 0)

	if format == "jpeg" || format == "png" || format == "gif" {
		return true
	}

	return false
}

func checkSize(bounds image.Rectangle) bool {
	return bounds.Dx() == width && bounds.Dy() == height
}

func resizeImg(img image.Image, bounds image.Rectangle) image.Image {
	var resized image.Image

	if bounds.Dx() > bounds.Dy() {
		resized = imaging.Resize(img, 0, height, imaging.Lanczos)
	} else {
		resized = imaging.Resize(img, width, 0, imaging.Lanczos)
	}

	cropped := imaging.CropCenter(resized, width, height)

	return cropped
}
