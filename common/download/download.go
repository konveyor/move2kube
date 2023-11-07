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
	"github.com/konveyor/move2kube-wasm/common"
	"github.com/sirupsen/logrus"
)

// DownloadOptions stores options for the downloader
type DownloadOptions struct {
	ContentURL              string
	DownloadDestinationPath string
	Overwrite               bool
}

// Downloader defines interface for downloaders
type Downloader interface {
	Download(DownloadOptions) (string, error)
}

// IsRemotePath checks if the provided string is a valid remote path or not
func IsRemotePath(str string) bool {
	return common.IsHTTPURL(str)
}

// GetDownloadedPath downloads the content using suitable downloader and then returns the downloaded file path
func GetDownloadedPath(contentURL string, downloadDestinationPath string, overwrite bool) string {
	var err error
	downloadedPath := ""
	if common.IsHTTPURL(contentURL) {
		content := HTTPContent{}
		downloadOpts := DownloadOptions{ContentURL: contentURL, DownloadDestinationPath: downloadDestinationPath, Overwrite: overwrite}
		downloadedPath, err = content.Download(downloadOpts)
		if err != nil {
			logrus.Fatalf("failed to download the content using http downloader. Error : %+v", err)
		}
	} else {
		logrus.Fatalf("other downloader backends are not currently supported.")
	}
	return downloadedPath
}
