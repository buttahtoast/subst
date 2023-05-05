#!/bin/sh

# Create Kubeconfig, if possible (CMP does not have access to the CLuster Kubernetes environment Variables, therefore we need to pass them in)
if [ -f "/etc/kubernetes/kubeconfig" ]; then
  echo "🦄 /etc/kubernetes/kubeconfig already present"
else 
  # Create Kubeconfig, if possible (CMP does not have access to the CLuster Kubernetes environment Variables, therefore we need to pass them in)
  TOKEN=""
  if [ -f "/var/run/secrets/kubernetes.io/serviceaccount/token" ]; then
    TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
  fi
  CA=""
  if [ -f "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt" ]; then
    CA=$(cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt | base64 -w0)
  fi
  if [ -z "$TOKEN" ] || [ -z "$CA" ]; then
    echo "💥 Unable to create Kubeconfig"
  else 
cat <<EOF > "/etc/kubernetes/kubeconfig"
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ${CA}
    server: https://kubernetes.default.svc
  name: default-cluster
contexts:
- context:
    cluster: default-cluster
    namespace: default
    user: default-auth
  name: default-context
current-context: default-context
kind: Config
preferences: {}
users:
- name: default-auth
  user:
    token: ${TOKEN}
EOF
    echo "🦄 Kubeconfig Created"
  fi
fi

exec "$@"