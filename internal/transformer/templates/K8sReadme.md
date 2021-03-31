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
{{- if .NewImages }}
{{- if .AddCopySources}}
* Copy the source directory into the "./source/" folder for packaging as containers using "./scripts/copysources.sh <SRC_DIR>"
{{- end }}
* Build your images using "./scripts/buildimages.sh"
* Push images to registry "./scripts/pushimages.sh <REGISTRY_URL> <REGISTRY_NAMESPACE>"
{{- end}}
* The k8s yamls are in "./deploy/yamls/". Use "./scripts/deploy.sh" to deploy them into a kubernetes cluster.
* The helm chart is at "./deploy/helm/". Use "./scripts/deployhelm.sh" to install it.
* The operator is at "./deploy/operator/".
