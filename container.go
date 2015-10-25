package chronos

type Volume struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
	Mode          string `json:"mode"`
}

type Parameter struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Container struct {
	Type    string `json:"type"`
	Image   string `json:"image"`
	Network string `json:"network"`

	Volumes []*Volume `json:"volume"`

	Parameters []*Parameter `json:"parameters"`

	ForcePullImage bool `json:"forcePullImage"`
}

func NewDockerContainer() *Container {
	return &Container{
		Type:    "DOCKER",
		Network: "BRIDGE",
	}
}
