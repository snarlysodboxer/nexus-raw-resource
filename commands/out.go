package commands

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	resource "github.com/snarlysodboxer/nexus-raw-resource"
)

type Out struct {
	stdin  io.Reader
	stderr io.Writer
	stdout io.Writer
	args   []string
}

func NewOut(stdin io.Reader, stderr, stdout io.Writer, args []string) *Out {
	return &Out{stdin: stdin, stderr: stderr, stdout: stdout, args: args}
}

// Execute does a PUT, and returns the version
func (o *Out) Execute() error {
	setupLogging(o.stderr)

	if len(o.args) < 2 {
		return fmt.Errorf("Arg 1 must specified, the directory containing the input")
	}

	// decode request
	var outRequest resource.OutRequest
	decoder := json.NewDecoder(o.stdin)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&outRequest)
	if err != nil {
		return fmt.Errorf("Invalid payload: %s", err)
	}

	// enable debug logging
	if outRequest.Source.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// validate params
	if err = outRequest.Params.Validate(); err != nil {
		return err
	}

	// enable insecure
	if outRequest.Source.Insecure {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		logrus.Warn("Insecure Transport enabled")
	}

	// read the file
	localPath := filepath.Join(o.args[1], outRequest.Params.File)
	logrus.Debugf("Reading file for upload: '%s'", localPath)
	data, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer data.Close()

	// set remotePath and fullURL
	remotePath := fmt.Sprintf("%s/%s",
		outRequest.Params.Group,
		filepath.Base(outRequest.Params.File),
	)
	logrus.Debugf("Remote path: '%s'", remotePath)
	fullURL := fmt.Sprintf("%s/repository/%s%s",
		outRequest.Source.URL,
		outRequest.Source.Repository,
		remotePath,
	)
	logrus.Debugf("URL for PUT: %s", fullURL)

	version, err := GetVersion(outRequest.Source, remotePath)
	if err != nil {
		return err
	}
	logrus.Debugf("Version capture match: '%s'", version)

	// do request
	req, err := http.NewRequest("PUT", fullURL, data)
	if err != nil {
		return err
	}
	req.SetBasicAuth(outRequest.Source.Username, outRequest.Source.Password)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// check response
	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("Unexpected status code uploading file: %d, %v", res.StatusCode, res)
	}

	// calculate md5 sum
	contents, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer contents.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, contents); err != nil {
		return err
	}
	sum := fmt.Sprintf("%x", hash.Sum(nil))

	// respond to Concourse
	err = json.NewEncoder(os.Stdout).Encode(resource.OutResponse{
		Version: resource.Version{
			Version: version,
			Path:    remotePath[1:],
			MD5Sum:  sum,
		},
		Metadata: []resource.MetadataField{
			{
				Name:  "url",
				Value: fullURL,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not encode JSON: %s", err)
	}

	return nil
}
