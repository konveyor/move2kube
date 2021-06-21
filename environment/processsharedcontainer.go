/*
Copyright IBM Corporation 2021

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

package environment

/*
type ProcessSharedContainer struct {
	Name string
	Pid  int

	TempPath string

	EnvContext string
	EnvSource  string
	OutContext string
	OutSource  string

	ImageName     string
	ImageWithData string
	CID           string // A started instance of ImageWithData
	PID           int
	Root          string
}

func NewProcessSharedContainer(name string, source string, context string, pid int) (EnvironmentInstance, error) {
	ei := ProcessSharedContainer{
		Name: name,

		Source:  source,
		Context: context,
	}
	return ei, nil
}

if env.Container.CID == "" {
	if env.Container.PID == 0 {
		env.OutContext, err = ioutil.TempDir(env.TempPath, types.AppNameShort)
		if err != nil {
			logrus.Errorf("Unable to create temp dir : %s", err)
		}
		if err := filesystem.Replicate(env.OutContext, env.EnvContext); err != nil {
			logrus.Errorf("Unable to copy contents to directory %s, dp: %s", env.OutSource, env.EnvSource, err)
		}
		env.OutSource, err = ioutil.TempDir(env.TempPath, workspaceDir)
		if err != nil {
			logrus.Errorf("Unable to create temp dir : %s", err)
		}
	} else {
		env.OutContext = env.makePathRelativeForContainer(filepath.Join(string(filepath.Separator), types.AppNameShort))
		env.OutSource = env.makePathRelativeForContainer(filepath.Join(string(filepath.Separator), workspaceDir))
	}
	if err := filesystem.Replicate(env.OutSource, env.EnvSource); err != nil {
		logrus.Errorf("Unable to copy contents to directory %s, dp: %s", env.OutSource, env.EnvSource, err)
	}
}

filepath.Join(string(filepath.Separator), "proc", fmt.Sprint(e.Container.PID), "root", types.AppNameShort)
	} else {



		func (e *Environment) Exec(cmd environmenttypes.Command) (string, string, int, error) {
			if workingDir == "" {
				workingDir = e.Context
			}
			if (e.Container != ContainerEnvironment{}) {
				if e.Container.CID != "" {
					cengine := GetContainerEngine()
					return cengine.RunCmdInContainer(e.Container.CID, cmd, "")
				}
				if e.Container.PID != 0 {
					//TODO : Fix me
					workingDir = filepath.Join("proc", fmt.Sprint(e.Container.PID), "root", workingDir)
				} else if workingDir == "" && e.Container.Root != "" {
					workingDir = e.Container.Root
				}
			}
			var exitcode int
			var outb, errb bytes.Buffer
			execcmd := exec.Command(cmd.CMD, cmd.Args...)
			execcmd.Dir = e.Context
			execcmd.Dir = workingDir
			execcmd.Stdout = &outb
			execcmd.Stderr = &errb
			err := execcmd.Run()
			if err != nil {
				var ee *exec.ExitError
				var pe *os.PathError
				if errors.As(err, &ee) {
					exitcode = ee.ExitCode()
					err = nil
				} else if errors.As(err, &pe) {
					logrus.Errorf("PathError during execution of command: %v", pe)
					err = pe
				} else {
					logrus.Errorf("Generic error during execution of command: %v", err)
				}
			}
			return outb.String(), errb.String(), exitcode, err
		}

*/
