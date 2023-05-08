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

type In struct {
	stdin  io.Reader
	stderr io.Writer
	stdout io.Writer
	args   []string
}

func NewIn(stdin io.Reader, stderr io.Writer, stdout io.Writer, args []string) *In {
	return &In{stdin: stdin, stderr: stderr, stdout: stdout, args: args}
}

func (i *In) Execute() error {
	setupLogging(i.stderr)

	if len(i.args) < 2 {
		return fmt.Errorf("Arg 1 must specified, the directory to place the file")
	}

	// decode request
	var inRequest resource.InRequest
	decoder := json.NewDecoder(i.stdin)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&inRequest)
	if err != nil {
		return fmt.Errorf("Invalid payload: %s", err)
	}

	// enable debug logging
	if inRequest.Source.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// enable insecure
	if inRequest.Source.Insecure {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		logrus.Warn("Insecure Transport enabled")
	}

	// set fullURL
	fullURL := fmt.Sprintf("%s/repository/%s/%s",
		inRequest.Source.URL,
		inRequest.Source.Repository,
		inRequest.Version.Path,
	)
	logrus.Debugf("URL for GET: %s", fullURL)

	inResponse := resource.InResponse{
		Version: inRequest.Version,
		Metadata: []resource.MetadataField{
			{
				Name:  "url",
				Value: fullURL,
			},
		},
	}

	// skip download
	if inRequest.Params.SkipDownload {
		err = json.NewEncoder(os.Stdout).Encode(inResponse)
		if err != nil {
			return fmt.Errorf("could not encode JSON: %s", err)
		}

		return nil
	}

	// do request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(inRequest.Source.Username, inRequest.Source.Password)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// check response
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status code downloading file: %d, %v", res.StatusCode, res)
	}

	// write to file
	localPath := filepath.Join(i.args[1], filepath.Base(inRequest.Version.Path))
	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, res.Body)
	if err != nil {
		return err
	}

	// ensure md5 sum matches
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
	if sum != inRequest.Version.MD5Sum {
		return fmt.Errorf("mismatched md5 sums! File: '%s', Version: '%s'", sum, inRequest.Version.MD5Sum)
	}

	// write metadata files
	fileContents := map[string]string{
		filepath.Join(i.args[1], "md5sum"):  sum,
		filepath.Join(i.args[1], "url"):     fullURL,
		filepath.Join(i.args[1], "version"): inRequest.Version.Version,
	}
	for file, contents := range fileContents {
		err = os.WriteFile(file, []byte(contents), 0644)
		if err != nil {
			return err
		}
	}

	// respond to Concourse
	err = json.NewEncoder(os.Stdout).Encode(inResponse)
	if err != nil {
		return fmt.Errorf("could not encode JSON: %s", err)
	}

	return nil
}
