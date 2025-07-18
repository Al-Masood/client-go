package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

func main() {
	kubeconfig := ""
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	kubeconfigFlag := flag.String("kubeconfig", kubeconfig, "Path to kubeconfig")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfigFlag)
	if err != nil {
		panic(err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	statefulSetRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}

	fmt.Println("For Creating Deployment")
	prompt()
	createStatefulSet(statefulSetRes, client)

	fmt.Println("For Updating Deployment")
	prompt()
	updateStatefulSet(statefulSetRes, client)

	fmt.Println("For Deleting Deployment")
	prompt()
	deleteStatefulSet(statefulSetRes, client)
}

func createStatefulSet(statefulSetRes schema.GroupVersionResource, client dynamic.Interface) {
	statefulSet := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "StatefulSet",
			"metadata": map[string]interface{}{
				"name": "book-server",
			},
			"spec": map[string]interface{}{
				"replicas": int64(2),
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "book-server",
					},
				},
				"serviceName": "book-server",
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "book-server",
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":            "book-server",
								"image":           "almasood/book-server:latest",
								"imagePullPolicy": "IfNotPresent",
								"ports": []interface{}{
									map[string]interface{}{
										"name":          "http",
										"protocol":      "TCP",
										"containerPort": int64(80),
									},
								},
								"args": []interface{}{
									"serve",
									"--port=3000",
									"--secret=secret",
									"--auth=true",
								},
							},
						},
					},
				},
			},
		},
	}

	fmt.Println("Creating StatefulSet...")
	result, err := client.Resource(statefulSetRes).Namespace(corev1.NamespaceDefault).Create(context.TODO(), statefulSet, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created StatefulSet %q.\n", result.GetName())
	fmt.Println()
}

func updateStatefulSet(statefulSetRes schema.GroupVersionResource, client dynamic.Interface) {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := client.Resource(statefulSetRes).Namespace(corev1.NamespaceDefault).Get(context.TODO(), "book-server", metav1.GetOptions{})
		if err != nil {
			return err
		}

		err = unstructured.SetNestedField(result.Object, int64(1), "spec", "replicas")
		if err != nil {
			return err
		}

		containers := []interface{}{
			map[string]interface{}{
				"name":            "book-server",
				"image":           "almasood/book-server:latest",
				"imagePullPolicy": "IfNotPresent",
				"ports": []interface{}{
					map[string]interface{}{
						"name":          "http",
						"protocol":      "TCP",
						"containerPort": int64(80),
					},
				},
			},
		}

		err = unstructured.SetNestedField(result.Object, containers, "spec", "template", "spec", "containers")
		if err != nil {
			return err
		}

		_, err = client.Resource(statefulSetRes).Namespace(corev1.NamespaceDefault).Update(context.TODO(), result, metav1.UpdateOptions{})
		return err
	})

	if retryErr != nil {
		panic(retryErr)
	}
	fmt.Println("StatefulSet updated.")
}

func deleteStatefulSet(statefulSetRes schema.GroupVersionResource, client dynamic.Interface) {
	list, err := client.Resource(statefulSetRes).Namespace(corev1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, d := range list.Items {
		replicas, found, err := unstructured.NestedInt64(d.Object, "spec", "replicas")
		if err != nil || !found {
			fmt.Printf("Replicas not found for StatefulSet %s: %v\n", d.GetName(), err)
			continue
		}
		fmt.Printf(" * %s (%d replicas)\n", d.GetName(), replicas)
	}

	policy := metav1.DeletePropagationForeground
	err = client.Resource(statefulSetRes).Namespace(corev1.NamespaceDefault).Delete(context.TODO(), "book-server", metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("StatefulSet deleted.")
}

func prompt() {
	fmt.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	fmt.Println()
}
