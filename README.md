[![Build Status](https://travis-ci.org/cloud-native-taiwan/labels-syncker.svg?branch=master)](https://travis-ci.org/cloud-native-taiwan/labels-syncker)
# Labels Syncker
Sync GitHub labels on repos in a GitHub organization based on a YAML config file.

## Setting config
The following is a typical config:

```yaml
cloud-native-taiwan:
  fork: false
  labels:
  - name: test-1
    color: 1fe34a
    description: for test
  - name: test-2
    color: d73a4a
    repositories:
    - jobs
```

## Usage
Use the following commands to sync labels on repos:

```sh
$ make
$ ./out/labels-syncker --token=<access_token> --config=labels.yml
```