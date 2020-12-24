Move2Kube
---------
Congratulations! Move2Kube has generated the necessary build artfiacts for moving all your application components to Knative. Using the artifacts in this directory you can deploy your application in a Knative instance

Prerequisites
-------------
* Docker
* Kubectl

Next Steps
----------
{{if .NewImages -}}
* Copy this directory into your base source directory, so that the scripts gets merged at the right contexts. 
* Build your images using ./scripts/buildimages.sh
* Push images to registry ./scripts/pushimages.sh
{{end -}}
* Use ./scripts/deploy.sh to deploy your artifacts into a knative.