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

type PeerContainter struct {
	Name     string
	TempPath string
	Children []Environment

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

/*

func (e *Local) Init(name string, source string, context string, container environmenttypes.Container) (err error) {

}
} else {
env.OutContext = env.makePathRelativeForContainer(filepath.Join(string(filepath.Separator), types.AppNameShort))
env.OutSource = env.makePathRelativeForContainer(filepath.Join(string(filepath.Separator), workspaceDir))
}
if err := filesystem.Replicate(env.OutSource, env.EnvSource); err != nil {
logrus.Errorf("Unable to copy contents to directory %s, dp: %s", env.OutSource, env.EnvSource, err)
}
}
return env, nil
}


cengine := GetContainerEngine()
if cengine == nil {
	return env, fmt.Errorf("no working container runtime found")
}
newImageName := container.Image + strings.ToLower(env.Name+uniuri.NewLen(5))
err := cengine.CopyDirsIntoImage(container.Image, newImageName, map[string]string{env.EnvSource: env.OutSource})
if err != nil {
	logrus.Debugf("Unable to create new container image with new data")
	if container.ContainerBuild.Context != "" {
		err = cengine.BuildImage(container.Image, container.ContainerBuild.Context, container.ContainerBuild.Dockerfile)
		if err != nil {
			logrus.Errorf("Unable to build new container image for %s : %s", container.Image, err)
			return env, err
		}
		err = cengine.CopyDirsIntoImage(container.Image, newImageName, map[string]string{env.EnvSource: env.OutSource})
		if err != nil {
			logrus.Errorf("Unable to copy paths to new container image : %s", err)
		}
	} else {
		return env, err
	}
}
containerEnvironment.ImageWithData = newImageName
cid, err := cengine.CreateContainer(newImageName)
if err != nil {
	logrus.Errorf("Unable to start container with image %s : %s", newImageName, cid)
	return env, err
}
containerEnvironment.CID = cid


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



if e.Container.CID != "" {
	cengine := GetContainerEngine()
	err := cengine.StopAndRemoveContainer(e.Container.CID)
	if err != nil {
		logrus.Errorf("Unable to delete image %s : %s", e.Container.ImageWithData, err)
	}
	cid, err := cengine.CreateContainer(e.Container.ImageWithData)
	if err != nil {
		logrus.Errorf("Unable to start container with image %s : %s", e.Container.ImageWithData, cid)
		return err
	}
	e.Container.CID = cid
} else {
	err := filesystem.Replicate(sp, dp)
	if err != nil {
		logrus.Errorf("Unable to remove directory %s : %s", dp, err)
	}
}
return nil

func (e *Local) Destroy() error {
	if e.Container.ImageWithData != "" {
		cengine := GetContainerEngine()
		err := cengine.RemoveImage(e.Container.ImageWithData)
		if err != nil {
			logrus.Errorf("Unable to delete image %s : %s", e.Container.ImageWithData, err)
		}
		err = cengine.StopAndRemoveContainer(e.Container.CID)
		if err != nil {
			logrus.Errorf("Unable to stop and remove container %s : %s", e.Container.CID, err)
		}
	} else {
		for _, dp := range e.paths {
			err := os.RemoveAll(dp)
			if err != nil {
				logrus.Errorf("Unable to remove directory %s : %s", dp, err)
			}
		}
	}
	for _, env := range e.Children {
		if err := env.Destroy(); err != nil {
			logrus.Errorf("Unable to destroy environment : %s", err)
		}
	}
	return nil
}


if e.Container.CID != "" {
	cengine := GetContainerEngine()
	err = cengine.CopyDirsFromContainer(e.Container.CID, map[string]string{path: output})
	if err != nil {
		logrus.Errorf("Unable to copy paths to new container image : %s", err)
	}
	return output, err
} else if e.Container.PID != 0 {
	nsp := filepath.Join("proc", fmt.Sprint(e.Container.PID), "root", path)
	err = filesystem.Replicate(nsp, output)
	if err != nil {
		logrus.Errorf("Unable to replicate in syncoutput : %s", err)
		return path, err
	}
	return output, nil
}*/
