## 0.1.0 (October 24th, 2020)

* Initial version
* Can append to the system-wide PEM or JKS keystores custom PKIs and then inject them to the trusted paths in containers
* Uses Init Containers for the generation of the new CA bundles which include the custom PKIs
* Support for annotations for specifiying the path to inject, init container image or configmap which contain the custom PKIs
