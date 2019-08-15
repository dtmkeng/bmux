package bmux

// Configuration represents the data in your config.json file.
type Configuration struct {
	Push  []string          `json:"push"`
	GZip  bool              `json:"gzip"`
	Ports PortConfiguration `json:"ports"`
}

// PortConfiguration lets you configure the ports that Aero will listen on.
type PortConfiguration struct {
	HTTP  int `json:"http"`
	HTTPS int `json:"https"`
}
