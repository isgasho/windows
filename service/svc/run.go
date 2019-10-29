package svc

type Service interface {
	Start(Logger) error
	Stop(Logger) error
}

type Logger interface {
	Info(string)
	Warn(string)
	Error(string)
}

func Run(s Service, name string, debug bool) error {
	return run(s, name, debug)
}
