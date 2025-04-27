package model

type Node struct {
	ID             string
	Name           string
	State          State
	HealthEndpoint string
	UpdEndpoint    string
}
