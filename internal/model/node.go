package model

import "net"

type Node struct {
	ID             string
	Name           string
	IpAddr         net.IP
	State          State
	HealthEndpoint string
	UpdEndpoint    string
}
