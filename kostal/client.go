package kostal

import (
	"fmt"
	"sort"

	"github.com/things-go/go-modbus"
)

type Client struct {
	c      modbus.Client
	serial string
	regs   []Register
}

func NewClient(tcpAddr string, verbose bool) (*Client, error) {
	opts := []modbus.ClientProviderOption{}
	if verbose {
		opts = append(opts, modbus.WithEnableLogger())
	}

	p := modbus.NewTCPClientProvider(tcpAddr, opts...)
	client := modbus.NewClient(p)
	err := client.Connect()
	if err != nil {
		return nil, fmt.Errorf("cound not connect to modbus: %w", err)
	}
	kc := &Client{c: client, regs: registers()}
	sn, _, err := kc.ReadField("inverter_serial_number")
	if err != nil {
		return nil, fmt.Errorf("could not read serial number from modbus: %w", err)
	}
	kc.serial = sn.(string)
	return kc, nil
}

func (c *Client) Close() error {
	return c.c.Close()
}

func (c *Client) SerialNumber() string {
	return c.serial
}

func (c *Client) ReadAddr(addr uint16) (any, Register, error) {
	for _, e := range c.regs {
		if e.Addr != addr {
			continue
		}
		v, err := e.Read(c.c)
		return v, e, err
	}
	return nil, Register{}, fmt.Errorf("unknown register address: %d", addr)
}

func (c *Client) ReadField(field string) (any, Register, error) {
	for _, e := range c.regs {
		if e.InfluxField != field {
			continue
		}
		v, err := e.Read(c.c)
		return v, e, err
	}
	return nil, Register{}, fmt.Errorf("unknown field: %s", field)
}

type ReadResult struct {
	Register Register
	Value    any
}

func (c *Client) ReadAll(filter func(Register) bool) ([]*ReadResult, error) {
	m := []*ReadResult{}
	for _, e := range c.regs {
		if !filter(e) {
			continue
		}
		v, err := e.Read(c.c)
		if err != nil {
			return nil, err
		}
		m = append(m, &ReadResult{Register: e, Value: v})
	}
	sort.Slice(m, func(i, j int) bool { return m[i].Register.Addr < m[j].Register.Addr })
	return m, nil
}
