package main

import (
	"fmt"
	"net"
	"strings"
)

type (
	Response struct {
		Host        string
		IPs         []net.IP
		MXrecord    []*net.MX
		CNAMErecord string
		Error       []error
	}

	ServerResponse struct {
		Host        string     `json:"host"`
		IPs         []string   `json:"ips,omitempty"`
		MXrecord    []MXrecord `json:"mx_records,omitempty"`
		CNAMErecord string     `json:"cname_record,omitempty"`
		Error       []error
	}
	MXrecord struct {
		Host string
		Pref uint16
	}
)

func (m Response) String() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "values for %s:\n", m.Host)
	if m.IPs != nil {
		fmt.Fprintf(&builder, "\tip(s): ")
	}
	for _, v := range m.IPs {
		fmt.Fprintf(&builder, "%s ", v.String())
	}
	builder.WriteByte('\n')
	for n, v := range m.MXrecord {
		if n > 0 {
			fmt.Fprintf(&builder, "\tMXrecords: host->%s, pref->%d\n", v.Host, v.Pref)
		}
	}

	if m.CNAMErecord != "" {
		fmt.Fprintf(&builder, "\tCNAME: %s\n", m.CNAMErecord)
	}

	if m.Error != nil {
		builder.WriteString("some errors occured during process:\n")
		for _, err := range m.Error {
			fmt.Fprintf(&builder, "\tErr: %v\n", err)
		}
	}
	return builder.String()
}
