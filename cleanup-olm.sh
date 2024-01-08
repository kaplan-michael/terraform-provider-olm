kubectl delete apiservices.apiregistration.k8s.io v1.packages.operators.coreos.com

kubectl delete crd catalogsources.operators.coreos.com
kubectl delete crd clusterserviceversions.operators.coreos.com
kubectl delete crd installplans.operators.coreos.com
kubectl delete crd olmconfigs.operators.coreos.com
kubectl delete crd operatorconditions.operators.coreos.com
kubectl delete crd operatorgroups.operators.coreos.com
kubectl delete crd operators.operators.coreos.com
kubectl delete crd subscriptions.operators.coreos.com

kubectl delete namespace operators
kubectl delete namespace olm

kubectl delete deployment packageserver
kubectl delete deployment catalog-operator
kubectl delete deployment olm-operator
