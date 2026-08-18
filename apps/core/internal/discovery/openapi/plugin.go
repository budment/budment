package openapi

// Plugin implements the workflow.PluginContract for the REST protocol.
type Plugin struct{}

func NewPlugin() *Plugin {
	return &Plugin{}
}

// Protocol explicitly identifies this plugin's domain.
func (p *Plugin) Protocol() string {
	return "rest"
}

// Priority decides conflicts. If two plugins define "get",
// the lower priority number wins (10 is high priority).
func (p *Plugin) Priority() int {
	return 10
}

// Operations returns the explicit Actions this plugin owns.
// When Workflow parses `- get: /users`, it asks the Registry,
// which maps "get" back to this "rest" plugin.
func (p *Plugin) Operations() []string {
	return []string{
		"get",
		"post",
		"put",
		"patch",
		"delete",
		"options",
		"head",
	}
}

// ReservedFields declares fields strictly owned by REST in the step block.
func (p *Plugin) ReservedFields() []string {
	return []string{"headers", "cookies"}
}
