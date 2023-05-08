package commands

import (
	"fmt"
	"regexp"

	resource "github.com/snarlysodboxer/nexus-raw-resource"
)

// GetVersion gets the matched version for a particular path
func GetVersion(source resource.Source, path string) (string, error) {
	regex, err := regexp.Compile(source.CaptureRegex)
	if err != nil {
		return "", err
	}
	matches := regex.FindStringSubmatch(path)
	if len(matches) == 2 {
		return matches[1], nil
	}
	if len(matches) < 2 {
		return "", fmt.Errorf("No match found for '%s' in '%s'", source.CaptureRegex, path)
	}

	return "", fmt.Errorf("Multiple matches found for '%s' in '%s'", source.CaptureRegex, path)
}
