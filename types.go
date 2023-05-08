package resource

import (
	"fmt"
	"strings"
)

type CheckRequest struct {
	Source  Source   `json:"source"`
	Version *Version `json:"version"`
}

type CheckResponse []Version

type InRequest struct {
	Source  Source    `json:"source"`
	Params  GetParams `json:"params"`
	Version Version   `json:"version"`
}

type InResponse struct {
	Version  Version         `json:"version"`
	Metadata []MetadataField `json:"metadata"`
}

type OutRequest struct {
	Source Source    `json:"source"`
	Params PutParams `json:"params"`
}

type OutResponse struct {
	Version  Version         `json:"version"`
	Metadata []MetadataField `json:"metadata"`
}

type MetadataField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Source struct {
	URL          string `json:"url"`
	Insecure     bool   `json:"insecure,omitempty"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Repository   string `json:"repository"`
	CaptureRegex string `json:"capture-regex"`
	Debug        bool   `json:"debug,omitempty"`
}

func (source Source) Validate() error {
	if source.URL == "" {
		return fmt.Errorf("url must be set")
	}

	if source.Username == "" {
		return fmt.Errorf("username must be set")
	}

	if source.Password == "" {
		return fmt.Errorf("password must be set")
	}

	if source.Repository == "" {
		return fmt.Errorf("repository must be set")
	}

	if source.CaptureRegex == "" {
		return fmt.Errorf("capture-regex must be set")
	}

	return nil
}

type Version struct {
	Version string `json:"version,omitempty"`
	Path    string `json:"path,omitempty"`
	MD5Sum  string `json:"md5sum,omitempty"`
}

type GetParams struct {
	SkipDownload bool `json:"skip_download"`
}

type PutParams struct {
	File  string `json:"file"`
	Group string `json:"group"`
}

func (params PutParams) Validate() error {
	if strings.HasPrefix(params.File, `/`) {
		return fmt.Errorf("file must be a relative path including the input name")
	}

	if !strings.HasPrefix(params.Group, `/`) {
		return fmt.Errorf("group must start with a slash")
	}

	return nil
}
