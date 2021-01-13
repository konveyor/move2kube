Move2Kube
---------
Congratulations! Move2Kube has generated the necessary build artifacts for moving all your application components to Knative. Using the artifacts in this directory you can deploy your application in a Knative instance

Prerequisites
-------------
* Docker
* Kubectl

Next Steps
----------
{{- if .NewImages }}
{{- if .AddCopySources }}
* Copy the source directory into the containers folder for packaging as containers using "./scripts/copysource.sh <SRC_DIR>"
{{- end }}
* Build your images using ./scripts/buildimages.sh
* Push images to registry ./scripts/pushimages.sh
{{- end }}
* Use ./scripts/deploy.sh to deploy your artifacts into a knative.
