/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package sshkeys

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	commonknownhosts "github.com/konveyor/move2kube/common/knownhosts"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const (
	userProvidedKey = "user_provided_key"
)

var (
	// DomainToPublicKeys maps domains to public keys gathered with known-hosts/get-public-keys.sh
	DomainToPublicKeys = map[string][]string{
		"github.com":    {"github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+PXYPCPy6rbTrTtw7PHkccKrpp0yVhp5HdEIcKr6pLlVDBfOLX9QUsyCOV0wzfjIJNlGEYsdlLJizHhbn2mUjvSAHQqZETYP81eFzLQNnPHt4EVVUh7VfDESU84KezmD5QlWpXLmvU31/yMf+Se8xhHTvKSCZIFImWwoG6mbUoWf9nzpIoaSjB+weqqUUmpaaasXVal72J+UX2B+2RPW3RcT0eOzQgqlJL3RKrTJvdsjE3JEAvGq3lGHSZXy28G3skua2SmVi/w4yCE6gbODqnTWlg7+wC604ydGXA8VJiS5ap43JXiUFFAaQ=="},
		"gitlab.com":    {"gitlab.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAfuCHKVTjquxvt6CM6tdG4SLp1Btn/nOeHHE5UOzRdf", "gitlab.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCsj2bNKTBSpIYDEGk9KxsGh3mySTRgMtXL583qmBpzeQ+jqCMRgBqB98u3z++J1sKlXHWfM9dyhSevkMwSbhoR8XIq/U0tCNyokEi/ueaBMCvbcTHhO7FcwzY92WK4Yt0aGROY5qX2UKSeOvuP4D6TPqKF1onrSzH9bx9XUf2lEdWT/ia1NEKjunUqu1xOB/StKDHMoX4/OKyIzuS0q/T1zOATthvasJFoPrAjkohTyaDUz2LN5JoH839hViyEG82yB+MjcFV5MU3N1l1QL3cVUCh93xSaua1N85qivl+siMkPGbO5xR/En4iEY6K2XPASUEMaieWVNTRCtJ4S8H+9", "gitlab.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBFSMqzJeV9rUzU4kWitGjeR4PWSa29SPqJ1fVkhtj3Hw9xjLVXVYrU9QlYWrOLXBpQ6KWjbjTDTdDkoohFzgbEY="},
		"bitbucket.org": {"bitbucket.org ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAubiN81eDcafrgMeLzaFPsw2kNvEcqTKl/VqLat/MaB33pZy0y3rJZtnqwR2qOOvbwKZYKiEO1O6VqNEBxKvJJelCq0dTXWT5pbO2gDXC6h6QDXCaHo6pOHGPUy+YBaGQRGuSusMEASYiWunYN0vCAI8QaXnWMXNMdFP3jHAJH0eDsoiGnLPBlBp4TNm6rYI74nMzgz3B9IikW4WVK+dc8KZJZWYjAuORU3jc1c/NPskD2ASinf8v3xnfXeukU0sJ5N6m5E8VLjObPEO+mN2t/FZTMZLiFqPWc/ALSqnMnnhwrNi2rbfg/rd/IpL8Le3pSBne8+seeFVBoGqzHM9yXw=="},
	}
	privateKeyDir                    = ""
	firstTimeLoadingKnownHostsOfUser = true
	firstTimeLoadingSSHKeysOfUser    = true
	privateKeysToConsider            = []string{}
)

// LoadKnownHostsOfCurrentUser loads the public keys from known_hosts
func LoadKnownHostsOfCurrentUser() {
	if !firstTimeLoadingKnownHostsOfUser {
		return
	}
	firstTimeLoadingKnownHostsOfUser = false
	usr, err := user.Current()
	if err != nil {
		logrus.Warn("Failed to get the current user. Error:", err)
		return
	}
	home := usr.HomeDir
	logrus.Debugf("Home directory: %q", home)
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
	logrus.Debugf("Looking in the known_hosts at path %q for public keys.", knownHostsPath)

	// Ask if we should look at ~/.ssh/known_hosts
	message := `The CI/CD pipeline needs access to the git repos in order to clone, build and push.
Move2Kube has public keys for github.com, gitlab.com, and bitbucket.org by default.
If any of the repos use ssh authentication we will need public keys in order to verify.
Do you want to load the public keys from your [%s]?:`
	ans := qaengine.FetchBoolAnswer(common.ConfigRepoLoadPubKey, fmt.Sprintf(message, knownHostsPath), []string{"No, I will add them later if necessary."}, false, nil)
	if !ans {
		logrus.Debug("Don't read public keys from known_hosts. They will be added later if necessary.")
		return
	}

	newKeys, err := commonknownhosts.ParseKnownHosts(knownHostsPath)
	if err != nil {
		logrus.Warnf("Failed to get public keys from the known_hosts file at path %q Error: %q", knownHostsPath, err)
		return
	}
	for domain, keys := range newKeys {
		if _, ok := DomainToPublicKeys[domain]; !ok {
			DomainToPublicKeys[domain] = keys
		}
	}
	logrus.Debug("DomainToPublicKeys:", DomainToPublicKeys)
}

func loadSSHKeysOfCurrentUser() {
	if !firstTimeLoadingSSHKeysOfUser {
		return
	}
	firstTimeLoadingSSHKeysOfUser = false
	usr, err := user.Current()
	if err != nil {
		logrus.Warn("Failed to get the current user. Error:", err)
		return
	}
	home := usr.HomeDir
	logrus.Debugf("Home directory: %q", home)
	privateKeyDir = filepath.Join(home, ".ssh")
	logrus.Debugf("Looking in ssh directory at path %q for keys.", privateKeyDir)

	// Ask whether to load private keys or provide own key
	options := []string{
		"Load private ssh keys from " + privateKeyDir,
		"Provide your own key",
		"No, I will add them later if necessary.",
	}
	message := `The CI/CD pipeline needs access to the git repos in order to clone, build and push.
If any of the repos require ssh keys you will need to provide them.
Select an option:`
	selectedOption := qaengine.FetchSelectAnswer(common.ConfigRepoLoadPrivKey, message, nil, "", options, nil)

	switch selectedOption {
	case options[0]:
		if err := loadKeysFromDirectory(privateKeyDir); err != nil {
			logrus.Warn("Can't load keys from directory. Error:", err)
			return
		}

	case options[1]:
		privateKeysToConsider = []string{userProvidedKey}

	default:
		logrus.Debug("Don't read private keys. They will be added later if necessary.")
		return
	}

}

func loadKeysFromDirectory(directory string) error {
	finfos, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("failed to read the SSH directory at path %q: %w", directory, err)
	}

	if len(finfos) == 0 {
		logrus.Warn("No key files were found in", directory)
		return nil
	}

	filenames := []string{}
	for _, finfo := range finfos {
		filenames = append(filenames, finfo.Name())
	}

	selectedFilenames := qaengine.FetchMultiSelectAnswer(
		common.ConfigRepoKeyPathsKey,
		fmt.Sprintf("These are the files found in %q. Select the keys to consider:", directory),
		[]string{"Select all the keys that give access to git repos."},
		filenames,
		filenames,
		nil,
	)

	if len(selectedFilenames) == 0 {
		logrus.Info("All key files ignored.")
		return nil
	}

	// Save the filenames for now. We will decrypt them if and when we need them.
	privateKeysToConsider = selectedFilenames
	return nil
}

func marshalRSAIntoPEM(key *rsa.PrivateKey) string {
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	PEMBlk := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}
	PEMBytes := pem.EncodeToMemory(PEMBlk)
	return string(PEMBytes)
}

func marshalECDSAIntoPEM(key *ecdsa.PrivateKey) string {
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		logrus.Errorf("Failed to marshal the ECDSA key. Error: %q", err)
		return ""
	}
	PEMBlk := &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}
	PEMBytes := pem.EncodeToMemory(PEMBlk)
	return string(PEMBytes)
}

func loadSSHKey(filename string) (string, error) {
	path := filepath.Join(privateKeyDir, filename)
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		logrus.Errorf("Failed to read the private key file at path %q Error: %q", path, err)
		return "", err
	}
	key, err := ssh.ParseRawPrivateKey(fileBytes)
	if err != nil {
		// Could be an encrypted private key.
		if _, ok := err.(*ssh.PassphraseMissingError); !ok {
			logrus.Errorf("Failed to parse the private key file at path %q Error %q", path, err)
			return "", err
		}

		qaKey := common.JoinQASubKeys(common.ConfigRepoPrivKey, `"`+filename+`"`, "password")
		desc := fmt.Sprintf("Enter the password to decrypt the private key %q : ", filename)
		hints := []string{"Password:"}
		password := qaengine.FetchPasswordAnswer(qaKey, desc, hints, nil)
		key, err = ssh.ParseRawPrivateKeyWithPassphrase(fileBytes, []byte(password))
		if err != nil {
			logrus.Errorf("Failed to parse the encrypted private key file at path %q Error %q", path, err)
			return "", err
		}
	}
	// *ecdsa.PrivateKey
	switch actualKey := key.(type) {
	case *rsa.PrivateKey:
		return marshalRSAIntoPEM(actualKey), nil
	case *ecdsa.PrivateKey:
		return marshalECDSAIntoPEM(actualKey), nil
	default:
		logrus.Errorf("Unknown key type [%T]", key)
		return "", fmt.Errorf("unknown key type [%T]", key)
	}
}

// GetSSHKey returns the private key for the given domain.
func GetSSHKey(domain string) (string, bool) {
	loadSSHKeysOfCurrentUser()
	if len(privateKeysToConsider) == 0 {
		return "", false
	}
	if privateKeysToConsider[0] == userProvidedKey {
		key := qaengine.FetchStringAnswer(common.ConfigRepoPrivKey, "Provide your own PEM-formatted private key:", []string{"Should not be empty"}, "", nil)
		if key == "" {
			logrus.Error("User-provided private key is empty.")
			return "", false
		}
		if err := validatePEMPrivateKey(key); err != nil {
			logrus.Error("Can't validate the PEM-formatted private key. Error:", err)
			return "", false
		}
		return key, true
	}

	filenames := privateKeysToConsider
	noAnswer := "none of the above"
	filenames = append(filenames, noAnswer)
	qaKey := common.JoinQASubKeys(common.ConfigRepoKeysKey, `"`+domain+`"`, "key")
	desc := fmt.Sprintf("Select the key to use for the git domain %s :", domain)
	hints := []string{fmt.Sprintf("If none of the keys are correct, select %s", noAnswer)}
	filename := qaengine.FetchSelectAnswer(qaKey, desc, hints, noAnswer, filenames, nil)
	if filename == noAnswer {
		logrus.Debugf("No key selected for domain %s", domain)
		return "", false
	}

	logrus.Debug("Loading the key", filename)
	key, err := loadSSHKey(filename)
	if err != nil {
		logrus.Warnf("Failed to load the key %q Error %q", filename, err)
		return "", false
	}
	return key, true
}

func validatePEMPrivateKey(key string) error {
	block, _ := pem.Decode([]byte(key))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return fmt.Errorf("invalid PEM private key format")
	}

	_, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	return nil
}
