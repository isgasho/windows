package settings

type Settings struct {
	Configuration    string
	ReportDeviceName bool
	CheckUpdates     bool
}

func FromMap(m map[string]interface{}) Settings {
	var s Settings
	if v, ok := m["configuration"].(string); ok {
		s.Configuration = v
	}
	if v, ok := m["reportDeviceName"].(bool); ok {
		s.ReportDeviceName = v
	}
	if v, ok := m["checkUpdates"].(bool); ok {
		s.CheckUpdates = !v
	}
	return s
}
