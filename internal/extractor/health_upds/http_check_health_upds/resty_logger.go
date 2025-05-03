package http_check_health_upds

type emptyRestyLogger struct{}

func (emptyRestyLogger) Errorf(format string, v ...any) {}
func (emptyRestyLogger) Warnf(format string, v ...any)  {}
func (emptyRestyLogger) Debugf(format string, v ...any) {}
