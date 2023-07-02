/*
 *  Copyright IBM Corporation 2023
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

package download

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

// HTTPContent stores remote content config
type HTTPContent struct {
	ContentFilePath string
}

// Download downloads content from the given content URL
func (content *HTTPContent) Download(downloadOptions DownloadOptions) (string, error) {
	if downloadOptions.DownloadDestinationPath == "" {
		return "", fmt.Errorf("the path where the content has to be downloaded is empty - %s", downloadOptions.DownloadDestinationPath)
	}

	_, err := os.Stat(downloadOptions.DownloadDestinationPath)
	if os.IsNotExist(err) {
		logrus.Debugf("downloaded content would be available at '%s'", downloadOptions.DownloadDestinationPath)
	} else if downloadOptions.Overwrite {
		logrus.Infof("downloaded content will be overwritten at %s", downloadOptions.DownloadDestinationPath)
		err = os.RemoveAll(downloadOptions.DownloadDestinationPath)
		if err != nil {
			return "", fmt.Errorf("failed to remove the directory at the given path - %s. Error : %+v", downloadOptions.DownloadDestinationPath, err)
		}
	} else {
		return downloadOptions.DownloadDestinationPath, nil
	}
	logrus.Infof("Downloading the content using http downloader into %s. This might take some time.", downloadOptions.DownloadDestinationPath)

	out, err := os.Create(downloadOptions.DownloadDestinationPath)
	if err != nil {
		return "", fmt.Errorf("failed to create a file for the provided path - %s. Error : %+v", downloadOptions.DownloadDestinationPath, err)
	}
	defer out.Close()

	resp, err := http.Get(downloadOptions.ContentURL)
	if err != nil {
		return "", fmt.Errorf("failed to http get content from the provided content url - %s. Error : %+v", downloadOptions.ContentURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to http get content from the provided content url - %s. Received status code %d", downloadOptions.ContentURL, resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to copy content from response to file. Error : %+v", err)
	}
	content.ContentFilePath = downloadOptions.DownloadDestinationPath
	return downloadOptions.DownloadDestinationPath, nil

}
