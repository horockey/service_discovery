package model

type Node struct {
	ID             string
	Hostname       string
	ServiceName    string
	State          State
	HealthEndpoint string
	UpdEndpoint    string
	Meta           map[string]string
}
