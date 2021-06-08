/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package knownhosts

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	markerCert    = "@cert-authority"
	markerRevoked = "@revoked"
)

func nextWord(line []byte) (string, []byte) {
	i := bytes.IndexAny(line, "\t ")
	if i == -1 {
		return string(line), nil
	}

	return string(line[:i]), bytes.TrimSpace(line[i:])
}

// parseKnownHostsLine parses the line to extract the hostname/IP.
// It also returns a bool to indicate if the line is a revocation or a certificate line and should be ignored.
func parseKnownHostsLine(line []byte) (shouldIgnore bool, host, lineStr string, err error) {
	lineStr = string(line)
	shouldIgnore = false

	if w, next := nextWord(line); w == markerCert || w == markerRevoked {
		shouldIgnore = true
		line = next
	}

	host, line = nextWord(line)
	if len(line) == 0 {
		return shouldIgnore, "", "", errors.New("knownhosts: missing host pattern")
	}

	// ignore the keytype as it's in the key blob anyway.
	_, line = nextWord(line)
	if len(line) == 0 {
		return shouldIgnore, "", "", errors.New("knownhosts: missing key type pattern")
	}

	keyBlob, _ := nextWord(line)
	keyBytes, err := base64.StdEncoding.DecodeString(keyBlob)
	if err != nil {
		return shouldIgnore, "", "", err
	}
	if _, err = ssh.ParsePublicKey(keyBytes); err != nil {
		return shouldIgnore, "", "", err
	}

	return shouldIgnore, host, lineStr, nil
}

// ParseKnownHosts creates a host key database from the given OpenSSH host key file.
// See the sshd manpage for more info: http://man.openbsd.org/sshd#SSH_KNOWN_HOSTS_FILE_FORMAT
func ParseKnownHosts(path string) (map[string][]string, error) {
	knownHostsFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer knownHostsFile.Close()

	domainToPublicKeys := map[string][]string{}
	scanner := bufio.NewScanner(knownHostsFile)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		shouldIgnore, hostName, lineStr, err := parseKnownHostsLine(line)
		if err != nil {
			return nil, fmt.Errorf("Error occurred parsing known_hosts file at path %q on line no. %d Error: %q", path, lineNum, err)
		}
		if shouldIgnore {
			continue
		}
		if hostName[0] == '|' {
			// TODO: not sure if we can support hashed hosts.
			continue
		}
		for _, h := range strings.Split(hostName, ",") {
			if len(h) == 0 {
				continue
			}
			domainToPublicKeys[h] = append(domainToPublicKeys[h], lineStr)
		}
	}

	return domainToPublicKeys, scanner.Err()
}

func addKeyToMap(hostTokey map[string]ssh.PublicKey) ssh.HostKeyCallback {
	return func(hostPort string, address net.Addr, key ssh.PublicKey) error {
		host, port, err := net.SplitHostPort(hostPort)
		if err != nil {
			logrus.Errorf("Failed to split %s into host and the port. Error %q", hostPort, err)
			return err
		}
		logrus.Debugf("host %s on port %s at address %v has the key %v of type %T", host, port, address, key, key)
		hostTokey[host] = key
		return nil
	}
}

// GetKey returns the ssh public key for the given host.
func GetKey(host string) ssh.PublicKey {
	// Setup
	defaultSSHPort := "22"
	defaultGitUser := "git"
	hostTokey := map[string]ssh.PublicKey{}
	config := &ssh.ClientConfig{
		User:            defaultGitUser,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: addKeyToMap(hostTokey),
	}
	url := net.JoinHostPort(host, defaultSSHPort)
	// Get key
	_, err := ssh.Dial("tcp", url, config)
	logrus.Debug(`This error should say "handshake failed: ssh: unable to authenticate, attempted methods [none], no supported methods remain". Error:`, err)
	if err == nil {
		logrus.Warnf("This should have failed but it didn't.")
	}
	return hostTokey[host]
}

// GetKnownHostsLine returns the line to be added to the known_hosts file.
// The line has the format: host <space> algo <space> base64 encoded public key
func GetKnownHostsLine(host string) (string, error) {
	key := GetKey(host)
	if key == nil {
		return "", fmt.Errorf("Failed to get the ssh public key for the host %s", host)
	}
	line := knownhosts.Line([]string{host}, key)
	return line, nil
}
