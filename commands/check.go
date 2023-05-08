package commands

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
	resource "github.com/snarlysodboxer/nexus-raw-resource"
)

type Check struct {
	stdin  io.Reader
	stderr io.Writer
	stdout io.Writer
	args   []string
}

func NewCheck(stdin io.Reader, stderr io.Writer, stdout io.Writer, args []string) *Check {
	return &Check{stdin: stdin, stderr: stderr, stdout: stdout, args: args}
}

// Execute does a Get, and returns the versions
func (c *Check) Execute() error {
	setupLogging(c.stderr)

	// decode request
	var checkRequest resource.CheckRequest
	decoder := json.NewDecoder(c.stdin)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&checkRequest)
	if err != nil {
		return fmt.Errorf("Invalid payload: %s", err)
	}

	// enable debug logging
	if checkRequest.Source.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// enable insecure
	if checkRequest.Source.Insecure {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		logrus.Warn("Insecure Transport enabled")
	}

	// get assets
	url := fmt.Sprintf("%s/service/rest/v1/assets?repository=%s",
		checkRequest.Source.URL,
		checkRequest.Source.Repository,
	)
	items, err := getAssetsRecursive(url, checkRequest.Source.Username, checkRequest.Source.Password, "")
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("got : %s", err)
	}

	// process items into versions
	versions := []resource.Version{}
	for _, item := range items {
		versionStr, err := GetVersion(checkRequest.Source, item.Path)
		if err != nil {
			// filter out items that don't match Source.CaptureRegex
			continue
		}

		version := resource.Version{
			Version: versionStr,
			Path:    item.Path[1:], // strip leading slash
			MD5Sum:  item.Checksum.MD5,
		}
		versions = append(versions, version)
	}

	// sort versions
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version < versions[j].Version
	})

	// do the Concourse 'newer versions' algorithm:
	//     https://concourse-ci.org/implementing-resource-types.html#resource-check
	checkResponse := resource.CheckResponse{}
	switch {
	case checkRequest.Version == nil:
		// the first time, version is unset, so return only the current (latest) version
		if len(versions) != 0 {
			checkResponse = resource.CheckResponse(versions[len(versions)-1:])
		}
	case checkRequest.Version != nil:
		// report only the version in the checkRequest, and any newer ones
		for _, version := range versions {
			if version.Version == checkRequest.Version.Version &&
				version.MD5Sum == checkRequest.Version.MD5Sum &&
				version.Path == checkRequest.Version.Path {
				checkResponse = append(checkResponse, version)
			}
			if version.Version > checkRequest.Version.Version {
				checkResponse = append(checkResponse, version)
			}
		}
		// if there's still no Versions in checkResponse, then we were unable to find a
		//     match for the checkRequest version, so return the current (latest) version
		if len(checkResponse) == 0 && len(versions) != 0 {
			checkResponse = resource.CheckResponse(versions[len(versions)-1:])
		}
	}

	// response to Concourse
	err = json.NewEncoder(os.Stdout).Encode(checkResponse)
	if err != nil {
		return fmt.Errorf("could not encode JSON: %s", err)
	}

	return nil
}

// AssetsResponse models the response from a GET to v1/assets
type AssetsResponse struct {
	Items             []Item `json:"items"`
	ContinuationToken string `json:"continuationToken"`
}

// Item represents a Nexus Asset
type Item struct {
	DownloadUrl string `json:"downloadUrl"`
	Path        string `json:"path"`
	ID          string `json:"id"`
	Repository  string `json:"repository"`
	Format      string `json:"format"`
	Checksum    struct {
		SHA512 string `json:"sha512"`
		SHA256 string `json:"sha256"`
		MD5    string `json:"md5"`
		SHA1   string `json:"sha1"`
	} `json:"checksum"`
	ContentType    string `json:"contentType"`
	LastModified   string `json:"lastModified"`
	LastDownloaded string `json:"lastDownloaded"`
	Uploader       string `json:"uploader"`
	UploaderIp     string `json:"uploaderIp"`
	FileSize       int    `json:"fileSize"`
}

// getAssetsRecursive follows pagination, getting all assets in the Nexus Repository
func getAssetsRecursive(url, username, password, continuationToken string) ([]Item, error) {
	u := url
	if continuationToken != "" {
		u = fmt.Sprintf("%s&continuationToken=%s", url, continuationToken)
	}
	logrus.Debugf("URL for GET: %s", u)

	// do request
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// check response
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code getting assets: %d, %v", res.StatusCode, res)
	}

	var aResponse AssetsResponse
	err = json.NewDecoder(res.Body).Decode(&aResponse)
	if err != nil {
		return nil, err
	}

	returnItems := aResponse.Items

	// get more if there are any
	if aResponse.ContinuationToken != "" {
		logrus.Debug("Got a continuationToken, requesting more items...")
		items, err := getAssetsRecursive(url, username, password, aResponse.ContinuationToken)
		if err != nil {
			return nil, err
		}
		returnItems = append(returnItems, items...)
	}

	return returnItems, nil
}
