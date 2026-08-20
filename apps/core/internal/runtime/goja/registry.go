package goja

// Registry stores the compiled JS source.
// New VMs load it once during initialization.
type HookRegistry struct {
	jsBundleCode string
}

func NewHookRegistry(bundle string) *HookRegistry {
	return &HookRegistry{
		jsBundleCode: bundle,
	}
}

func (r *HookRegistry) GetCode() string {
	return r.jsBundleCode
}
