package main

import (
	"reflect"
)

/* Generic Packet */

type pkt struct {
	header pktHead
	body pktBody
}

type pktHead struct {
	dest []byte
	src []byte
	ethType []byte
}

type pktBody struct {
	start interface{}
	data []byte
}

/* Row Packet */

type startRow struct {
	rowNum []byte
	pxlOffset []byte
	pxlCount []byte
	padding []byte
}


/* Methods */

func (p *pkt) make(pktType string, rowNum *int, rowData *[]byte, sendTo []byte, sendFrom []byte) []byte {
	p.header.dest = sendTo
	p.header.src = sendFrom

	// Construct initialisation packet
	if pktType == "initial" {
		p.header.ethType = []byte{0x1, 0x07}

		p.body.start = nil

        level := byte(brightness.lin())

        start := make([]byte, 21)
        power := []byte{level} // Overall brightness
        middle := []byte{0x05, 0x00}
        channels := []byte{level, level, level}
        end := make([]byte, 67)

		p.body.data = join(start, power, middle, channels, end)

	// Construct brightness packet
	} else if pktType == "brightness" {
        level := byte(brightness.rgb())

		p.header.ethType = []byte{0x0a, level}

		p.body.start = nil
		p.body.data = join([]byte{level, level, 0xff}, make([]byte, 63))

	// Construct row packet
	} else if pktType == "row" {
		p.header.ethType = []byte{0x55, 0x00}

		p.body.start = startRow{
			rowNum: []byte{byte(*rowNum)},
			pxlOffset: make([]byte, 2),
			pxlCount: []byte{0x00, byte(width)},
			padding: []byte{0x08, 0x88},
		}

		p.body.data = *rowData
	}

	return p.splice(p)
}

func (p *pkt) splice(packet *pkt) []byte {
	var output []byte

	head := p.header
	start := p.body.start
	data := p.body.data

	output = subjoin(head)

	if start != nil {
		output = join(output, subjoin(start))
	}

    return join(output, data)
}

/* Helpers */

func join(slices ...[]byte) []byte {
	length := 0
	for _, s := range slices {
		length += len(s)
	}

	joined := make([]byte, length)

	offset := 0
	for _, s := range slices {
		if s != nil {
			copy(joined[offset:], s)
			offset += len(s)
		}
	}

	return joined
}

func subjoin(slice interface{}) []byte {
	var joined []byte

	val := reflect.ValueOf(slice)
	num := val.NumField()

	for i := 0; i < num; i++ {
		field := val.Field(i).Bytes()
		joined = join(joined, field)
	}

	return joined
}
