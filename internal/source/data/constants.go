// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots at
// 2020-11-26 22:17:12.32425 +0530 IST m=+0.001915380

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

package data

const (

	Cfbuildpacks_yaml = `kind: cfcontainerizers
buildpackcontainerizers:
  - buildpackname: binary_buildpack
    containerbuildtype: cnb
    targetoptions:
      - cloudfoundry/cnb:cflinuxfs3
  - buildpackname: java_buildpack_offline
    containerbuildtype: cnb
    targetoptions:
      - cloudfoundry/cnb:cflinuxfs3
  - buildpackname: hwc_buildpack
    containerbuildtype: cnb
    targetoptions:
      - cloudfoundry/cnb:cflinuxfs3
  - buildpackname: dotnet_core_buildpack
    containerbuildtype: cnb
    targetoptions:
      - cloudfoundry/cnb:cflinuxfs3
`

)