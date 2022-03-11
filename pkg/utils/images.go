package utils

import "os"

type ComplianceComponent uint

const (
	OPENSCAP = iota
	OPERATOR
	CONTENT
)

var componentDefaults = []struct {
	defaultImage string
	envVar       string
}{
	{"quay.io/pravin_dsilva/openscap-ocp:1.3.5", "RELATED_IMAGE_OPENSCAP"},
	{"quay.io/pravin_dsilva/compliance-operator:latest", "RELATED_IMAGE_OPERATOR"},
	{"quay.io/pravin_dsilva/ocp4:latest", "RELATED_IMAGE_PROFILE"},
}

// GetComponentImage returns a full image pull spec for a given component
// based on the component type
func GetComponentImage(component ComplianceComponent) string {
	comp := componentDefaults[component]

	imageTag := os.Getenv(comp.envVar)
	if imageTag == "" {
		imageTag = comp.defaultImage
	}
	return imageTag
}
