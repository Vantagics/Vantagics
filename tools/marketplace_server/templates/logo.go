package templates

import "html/template"

// LogoURL is the versioned URL for the marketplace logo.
// Set by main() before calling InitTemplates().
// Default fallback is the unversioned path.
var LogoURL = "/marketplace-logo.png"

// BaseFuncMap provides the logoURL function shared by all templates.
var BaseFuncMap = template.FuncMap{
	"logoURL": func() string { return LogoURL },
}
