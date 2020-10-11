#!/bin/bash

# Create a configmap containing OpenShift CA
cat <<EOF | oc apply -f -
kind: ConfigMap
apiVersion: v1
metadata:
  name: service-cacert
  annotations:
    service.beta.openshift.io/inject-cabundle: 'true'
EOF

# Extract OpenShift CA and base64
caBundle=$(oc get cm service-cacert -o jsonpath='{.data.service-ca\.crt}'|base64)

# Update the MutatingWebhookConfig
oc patch "${kind:-mutatingwebhookconfiguration}" custom-ca-injector-pki --type='json' -p "[{'op': 'add', 'path': '/webhooks/0/clientConfig/caBundle', 'value':'${caBundle}'}]"
oc patch "${kind:-mutatingwebhookconfiguration}" custom-ca-injector-1 --type='json' -p "[{'op': 'add', 'path': '/webhooks/0/clientConfig/caBundle', 'value':'${caBundle}'}]"
