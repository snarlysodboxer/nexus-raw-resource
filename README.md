# Concourse Resource for Nexus Raw Repositories

## Features
* Versions based on Golang regex match.
* Supports downloading from/uploading to dynamic directories.

## Source Configuration

* `url`: *Required* The URL of the Nexus server

* `insecure`: *Optional* Disable TLS validation

* `username`: *Required* Basic Auth Username

* `password`: *Required* Basic Auth Password

* `repository`: *Required* The name of the Nexus Raw Repository

* `capture-regex`: *Required* The pattern used to match Nexus Assets (files) that should become a Concourse version, and to extract a string from file names representing the version number.

* `debug`: *Optional* Enable debug logging

## Behavior

### `check`: Find and extract versions from the Nexus repository

An artifact's Path will need to match `source.capture-regex` in order to become a version.

### `in`: Fetch a version from the repository

Places the following files in the destination:

* `(filename)`: The file fetched from the repository.

* `md5sum`: A file containing the MD5 sum of the file

* `url`: A file containing the URL to the file

* `version`: A file containing the version matched in the file name

#### Parameters

* `skip_download`: *Optional.* Defaults to `false`. Skip downloading object from
  Nexus. Value need to be a true/false string.

### `out`: Upload a file to the repository

#### Parameters

* `file`: *Required* Local relative path to the file to upload

* `group`: *Required* The directory in the Respository in which the file should be uploaded. Must start with a `/`.

## Example

``` yaml
---
resource_types:
- name: nexus
  type: registry-image
  source:
    repository: snarlysodboxer/nexus-raw-resource

resources:
- name: release
  type: nexus
  source:
    url: http://127.0.0.1
    repository: repositoryName
    regexp: path/to/release-(.*).tgz

jobs:
- name: test-get-and-push-nexus
  plan:
  # example getting a previously created file
  - get: release
    # do something with release/release-1.2.3.tgz

  # example uploading a file that was just created
  - put: release
    params:
      file: some-output/release-1.2.3.tgz
      group: /path/to/1.2.3
      # resulting file will be at /path/to/1.2.3/release-1.2.3.tgz
```

## Developing on this resource
### Local testing
* Copy the `*.example.json` files to their repective non-example versions.
* Edit the contents as needed.
* Check: `go run cmd/check/main.go < check.json | jq`
* In: `go run cmd/in/main.go $PWD/output < in.json | jq`
* Out: `go run cmd/out/main.go $PWD < out.json | jq`
