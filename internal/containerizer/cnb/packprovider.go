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

package cnb

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/konveyor/move2kube/types"
	log "github.com/sirupsen/logrus"
)

const (
	dockersock = "/var/run/docker.sock"
)

type packProvider struct {
}

func (p *packProvider) isAvailable() bool {
	_, err := exec.LookPath("pack")
	if err != nil {
		log.Debugf("Unable to find pack : %s", err)
		return false
	}
	_, err = os.Stat(dockersock)
	if os.IsNotExist(err) {
		log.Debugf("Unable to find pack docker socket, ignoring CNB based containerization approach : %s", err)
		return false
	}
	return true
}

func (p *packProvider) isBuilderSupported(path string, builder string) (bool, error) {
	if !p.isAvailable() {
		return false, errors.New("Pack not supported in this instance")
	}
	cmd := exec.Command("pack", "build", types.AppNameShort+"testcflinuxf2selector:1", "-B", builder, "-p", path)

	var wg sync.WaitGroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("RunCommand: cmd.StdoutPipe(): %v", err)
		return false, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Errorf("RunCommand: cmd.StderrPipe(): %v", err)
		return false, err
	}

	if err := cmd.Start(); err != nil {
		log.Errorf("RunCommand: cmd.Start(): %v", err)
		return false, err
	}

	outch := make(chan string, 10)

	scannerStdout := bufio.NewScanner(stdout)
	scannerStdout.Split(bufio.ScanLines)
	wg.Add(1)
	go func() {
		for scannerStdout.Scan() {
			text := scannerStdout.Text()
			if strings.TrimSpace(text) != "" {
				outch <- text
			}
		}
		wg.Done()
	}()
	scannerStderr := bufio.NewScanner(stderr)
	scannerStderr.Split(bufio.ScanLines)
	wg.Add(1)
	go func() {
		for scannerStderr.Scan() {
			text := scannerStderr.Text()
			if strings.TrimSpace(text) != "" {
				outch <- text
			}
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(outch)
	}()

	for t := range outch {
		log.Debug(t)
		if strings.Contains(t, "===> ANALYZING") {
			log.Debugf("Found compatible cnb for %s", path)
			_ = cmd.Process.Kill()
			return true, nil
		}
		if strings.Contains(t, "No buildpack groups passed detection.") {
			log.Debugf("No compatible cnb for %s", path)
			_ = cmd.Process.Kill()
			return false, nil
		}
	}
	return false, errors.New("Error while using pack")
}

func (p *packProvider) getAllBuildpacks(builders []string) (map[string][]string, error) { //[Containerization target option value] buildpacks
	buildpacks := map[string][]string{}
	log.Debugf("Getting data of all builders %s", builders)
	for _, builder := range builders {
		cmd := exec.Command("pack", "inspect-builder", string(builder))
		var buildpackregex = regexp.MustCompile(`(?s)Group\s#\d+:[\r\n\s]+[^\s]+`)
		outputStr, err := cmd.Output()
		log.Debugf("Builder %s data :%s", builder, outputStr)
		if err != nil {
			log.Warnf("Error while getting supported buildpacks for builder %s : %s", builder, err)
			continue
		}
		buildpackmatches := buildpackregex.FindAllString(string(outputStr), -1)
		log.Debugf("Builder %s data :%s", builder, buildpackmatches)
		for _, buildpackmatch := range buildpackmatches {
			buildpackfields := strings.Fields(buildpackmatch)
			buildpacks[builder] = append(buildpacks[builder], buildpackfields[len(buildpackfields)-1])
		}
	}
	return buildpacks, nil
}
