package svc

func Start(name string) error {
	return start(name)
}

func Stop(name string) error {
	return stop(name)
}
