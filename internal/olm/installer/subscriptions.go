package installer

import (
	"context"
	"fmt"
	olmresourceclient "github.com/kaplan-michael/terraform-provider-olm/internal/olm/client"
	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func (c Client) InstallOperator(ctx context.Context, resources []unstructured.Unstructured) (*olmresourceclient.Status, error) {

	subscriptions := filterResources(resources, func(r unstructured.Unstructured) bool {
		return r.GroupVersionKind() == schema.GroupVersionKind{
			Group:   olmapiv1alpha1.GroupName,
			Version: olmapiv1alpha1.GroupVersion,
			Kind:    olmapiv1alpha1.SubscriptionKind,
		}
	})

	log.Print("Creating subscription resources")
	objs := toObjects(subscriptions...)
	if err := c.DoCreate(ctx, objs...); err != nil {
		return nil, fmt.Errorf("failed to create subscriptions: %v", err)
	}

	for _, sub := range subscriptions {
		subscriptionKey := types.NamespacedName{Namespace: sub.GetNamespace(), Name: sub.GetName()}
		log.Printf("Waiting for subscription/%s to install CSV", subscriptionKey.Name)
		csvKey, err := c.getSubscriptionCSV(ctx, subscriptionKey)
		if err != nil {
			return nil, fmt.Errorf("subscription/%s failed to install CSV: %v", subscriptionKey.Name, err)
		}
		log.Printf("Waiting for clusterserviceversion/%s to reach 'Succeeded' phase", csvKey.Name)
		if err := c.DoCSVWait(ctx, csvKey); err != nil {
			return nil, fmt.Errorf("clusterserviceversion/%s failed to reach 'Succeeded' phase",
				csvKey.Name)
		}

	}
	status := c.GetObjectsStatus(ctx, objs...)
	return &status, nil

}

func (c Client) GetSubscriptionStatus(ctx context.Context, resources []unstructured.Unstructured) (*olmresourceclient.Status, error) {

	subscriptions := filterResources(resources, func(r unstructured.Unstructured) bool {
		return r.GroupVersionKind() == schema.GroupVersionKind{
			Group:   olmapiv1alpha1.GroupName,
			Version: olmapiv1alpha1.GroupVersion,
			Kind:    olmapiv1alpha1.SubscriptionKind,
		}
	})

	for _, sub := range subscriptions {
		subscriptionKey := types.NamespacedName{Namespace: sub.GetNamespace(), Name: sub.GetName()}
		log.Printf("Found CSV for subscription/%s", subscriptionKey.Name)
		csvKey, err := c.getSubscriptionCSV(ctx, subscriptionKey)
		if err != nil {
			return nil, fmt.Errorf("Can't find CSV for subscription/%s  %v", subscriptionKey.Name, err)
		}
		log.Printf("Check if CSV for subscription/%s is in the 'Succeeded' phase", csvKey.Name)
		if err := c.DoCSVWait(ctx, csvKey); err != nil {
			return nil, fmt.Errorf("clusterserviceversion/%s failed is not in the 'Succeeded' phase, please check the cluster",
				csvKey.Name)
		}
	}

	objs := toObjects(subscriptions...)

	status := c.GetObjectsStatus(ctx, objs...)
	installed, err := status.HasInstalledResources()
	if err != nil {
		return nil, fmt.Errorf("the Operator installation has resource errors: %v", err)
	} else if !installed {
		return nil, fmt.Errorf("the Operator is not installed: %v", err)
	}
	return &status, nil
}

func (c Client) GetSubscriptionResources(name, namespace, channel, operatorName, source,
	sourceNamespace, installPlanApproval string) ([]unstructured.Unstructured, error) {

	// build the subscription manifest from the plan
	subscription := &olmapiv1alpha1.Subscription{

		TypeMeta: metav1.TypeMeta{
			APIVersion: olmapiv1alpha1.SubscriptionCRDAPIVersion, // Set APIVersion
			Kind:       olmapiv1alpha1.SubscriptionKind,          // Set Kind
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: &olmapiv1alpha1.SubscriptionSpec{
			Channel:                channel,
			Package:                name,
			CatalogSource:          source,
			CatalogSourceNamespace: sourceNamespace,
			InstallPlanApproval:    olmapiv1alpha1.Approval(installPlanApproval),
		},
	}
	// Convert the subscription to unstructured format
	unstructuredSub, err := runtime.DefaultUnstructuredConverter.ToUnstructured(subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Subscription to unstructured format: %v", err.Error())
	}

	// Create a slice of unstructured resources and append the subscription
	var resources []unstructured.Unstructured
	resources = append(resources, unstructured.Unstructured{Object: unstructuredSub})

	return resources, nil
}

func (c Client) UninstallOperator(ctx context.Context, resources []unstructured.Unstructured) error {

	subscriptions := filterResources(resources, func(r unstructured.Unstructured) bool {
		return r.GroupVersionKind() == schema.GroupVersionKind{
			Group:   olmapiv1alpha1.GroupName,
			Version: olmapiv1alpha1.GroupVersion,
			Kind:    olmapiv1alpha1.SubscriptionKind,
		}
	})

	objs := toObjects(subscriptions...)

	status := c.GetObjectsStatus(ctx, objs...)
	installed, err := status.HasInstalledResources()
	if !installed && err == nil {
		return fmt.Errorf("the Operator is not installed: %v", err)
	}

	var Csvs []unstructured.Unstructured
	for _, sub := range subscriptions {
		subscriptionKey := types.NamespacedName{Namespace: sub.GetNamespace(), Name: sub.GetName()}
		csvKey, err := c.getSubscriptionCSV(ctx, subscriptionKey)
		if err != nil {
			return fmt.Errorf("couln't get subscriptions/%s CSV: %v", subscriptionKey.Name, err)
		}
		csv := olmapiv1alpha1.ClusterServiceVersion{}
		err = c.Client.KubeClient.Get(ctx, csvKey, &csv)
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Printf("couln't get CSV/%s CSV: %v", csvKey.Name, err)
				continue

			}

		}
		csv.APIVersion = olmapiv1alpha1.ClusterServiceVersionAPIVersion
		csv.Kind = olmapiv1alpha1.ClusterServiceVersionKind

		// Convert the csv to unstructured format
		unstructuredCsv, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&csv)
		if err != nil {
			return fmt.Errorf("failed to convert CSV to unstructured format: %v", err.Error())
		}
		// Append to a slice of unstructured csvs
		Csvs = append(Csvs, unstructured.Unstructured{Object: unstructuredCsv})
	}

	objs = append(objs, toObjects(Csvs...)...)
	return c.DoDelete(ctx, objs...)

}
