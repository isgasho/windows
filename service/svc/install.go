package svc

func Install(name, displayName, desc string) error {
	return install(name, displayName, desc)
}

func Remove(name string) error {
	return remove(name)
}
