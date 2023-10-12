package kostal

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	modbus "github.com/things-go/go-modbus"
)

type Register struct {
	Addr        uint16
	Description string
	format      string
	length      uint16
	InfluxField string
	Include     bool
}

func (r *Register) String() string {
	return fmt.Sprintf("0x%X - %s [%s, %d]", r.Addr, r.Description, r.format, r.length)
}

func (r *Register) Read(client modbus.Client) (any, error) {
	data, err := client.ReadHoldingRegistersBytes(71, r.Addr, r.length)
	if err != nil {
		return nil, err
	}
	switch r.format {
	case "Float":
		//so byte order is BigEndias .. but word order is Little Endian ?!
		return math.Float32frombits(binary.BigEndian.Uint32([]byte{data[2], data[3], data[0], data[1]})), nil
	case "U32":
		return binary.BigEndian.Uint32(data), nil
	case "U16":
		return binary.BigEndian.Uint16(data), nil
	case "S16":
		return int16(binary.BigEndian.Uint16(data)), nil
	case "String":
		{
			for i := range data {
				if data[i] == 0 {
					data[i] = ' '
				}
			}
			s := string(data)
			return strings.TrimSpace(s), nil
		}
	case "Bool":
		return binary.BigEndian.Uint16(data)&0x1 > 0, nil
	case "-":
		return nil, nil
	default:
		return nil, fmt.Errorf("ERROR - unknown conversion from %s", r.format)
	}

}
