Move2Kube
---------
Congratulations! Move2Kube has generated the necessary build artfiacts for moving all your application components to Kubernetes. Using the artifacts in this directory you can deploy your application in a kubernetes cluster.

Prerequisites
-------------
* Docker
* Helm
* Kubectl
* Source-To-Image (S2I) https://github.com/openshift/source-to-image

Next Steps
----------
{{if .NewImages -}}
* Copy the source directory into the containers folder for packaging as containers using "./scripts/copysource.sh <SRC_DIR>"
* Build your images using "./scripts/buildimages.sh"
* Push images to registry "./scripts/pushimages.sh <REGISTRY_URL> <REGISTRY_NAMESPACE>"
{{end -}}
{{- if .Helm -}}* Your helm chart is at {{ .Project }}, you can install it using "./scripts/helminstall.sh" or you can use the operator.{{- else -}}
* Use "deploy.sh" to deploy your artifacts into a kubernetes cluster.
{{- end}}
{{if .AddCopySourcesWarning }}
IMPORTANT!!: If you used the UI for translation then "./scripts/copysources.sh" may copy to the wrong folder.
This is a limitation of the beta version. It should be fixed in future versions of move2kube-ui.  
In the meantime you can either:
- copy the sources to the appropriate folders inside "./containers/" manually. "./containers/" has the same folder structure as the sources folder so simply copy the appropriate source files/folders to the corresponding folders inside "./containers/".
- move the sources into a directory with the same name and then try copysources with that.  
  Example: if sources is a folder called "foo" you might try moving it into "foo/foo/foo" and then doing "./scripts/copysources.sh path/to/foo/". This will require you to read "./scripts/copysource.sh" and have some knowledge of how "cp -r" works in order to get it right.
{{ end }}