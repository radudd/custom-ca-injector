package mutate

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value, omitempty"`
}

type injection struct {
	injectPem bool
	injectJks bool
}