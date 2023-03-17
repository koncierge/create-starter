package main

import (
	"log"
	"syscall"
	"time"
)

func streamImg(data [][]uint32, duration int) {
	timer := time.NewTimer(time.Duration(duration) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return
		default:
			start := time.Now()

			sendFrame(data)
			time.Sleep(getInterval(time.Since(start)))
		}
	}
}

func sendFrame(data [][]uint32) {
	sendTo := dest
	sendFrom := src

	if ifacePwm {
		sendFrom = srcPwm
	}

	// Send initialisation packet
	packet := pkt{}
	sendPkt(packet.make("initial", nil, nil, sendTo, sendFrom), 1, 1)

	// Set brightness (only required as needed)
	sendPkt(packet.make("brightness", nil, nil, sendTo, sendFrom), 1, 2)

	// Send pixel data
	for i, rowData := range data {

		// Add gamma correction and adjust brightness for current row
		adjusted := transform(rowData, 1)

		packet = pkt{}
		sendPkt(packet.make("row", &i, &adjusted, sendTo, sendFrom), 1, 2)
	}

	if output == "dual" {
		if auxPwm {
			sendFrom = srcPwm
		}

		subimage := mask(data)

		// Send initial packet(s), as above
		packet := pkt{}
		sendPkt(packet.make("initial", nil, nil, sendTo, sendFrom), 2, 1)

		sendPkt(packet.make("brightness", nil, nil, sendTo, sendFrom), 2, 2)

		// Send pixel data
		for i, rowData := range subimage {

			// Apply transformations, as above
			adjusted := transform(rowData, multiplier)

			packet = pkt{}
			sendPkt(packet.make("row", &i, &adjusted, sendTo, sendFrom), 2, 2)
		}
	}
}

func sendPkt(packet []byte, iface int, times int) {
	// iface options
	// 0 = all
	// 1 = primary iface only
	// 2 = aux iface only

	for i := 0; i < times; i++ {

		if iface == 0 || iface == 1 {
			err := syscall.Sendto(sock, packet, 0, &addr)
			if err != nil {
				log.Fatal("[FATAL] ", err)
			}
		}

		if iface == 0 || iface == 2 {
			err = syscall.Sendto(auxSock, packet, 0, &auxAddr)
			if err != nil {
				log.Fatal("[FATAL] ", err)
			}

		}
	}
}

/* Helpers */

func transform(row []uint32, multiplier float64) []byte {
	var adjusted []byte

	for _, px := range row {
		adjusted = append(adjusted, i32tob(power(px, multiplier)))
	}

	return adjusted
}

func mask(data [][]uint32) [][]uint32 {
	var subimage [][]uint32
	rowStart := maskY
	rowEnd := minInt(maskY+maskH, len(data))

	if rowStart >= rowEnd {
		return subimage
	}

	for _, rowData := range data[rowStart:rowEnd] {
		channelStart := maskX * 3
		channelEnd := minInt((maskX+maskW)*3, len(rowData))

		if channelStart >= channelEnd {
			subimage = append(subimage, []uint32{})
			continue
		}

		subrow := make([]uint32, channelEnd-channelStart)
		copy(subrow, rowData[channelStart:channelEnd])
		subimage = append(subimage, subrow)
	}

	return subimage
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}

	return b
}

func getInterval(offset time.Duration) time.Duration {
	interval := int64(1000000 / fps)
	sleep := interval - offset.Microseconds()
	if sleep < 0 {
		return 0
	}

	return time.Duration(sleep * int64(time.Microsecond))
}
