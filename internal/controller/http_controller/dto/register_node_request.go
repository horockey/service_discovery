package dto

type RegisterNodeRequest struct {
	Hostname       string
	ServiceName    string
	HealthEndpoint string
	UpdEndpoint    string
}
