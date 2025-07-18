package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"os"
	"path/filepath"
)

func main() {
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "")
	}

	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	statefulSetsClient := clientset.AppsV1().StatefulSets(corev1.NamespaceDefault)

	fmt.Println("For Creating Deployment")
	prompt()
	createStatefulSet(statefulSetsClient)

	fmt.Println("For Updating Deployment")
	prompt()
	updateStatefulSet(statefulSetsClient)

	fmt.Println("For Deleting Deployment")
	prompt()
	deleteStatefulSet(statefulSetsClient)

}

func createStatefulSet(statefulSetsClient v1.StatefulSetInterface) {
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "book-server",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    ptr.To[int32](2),
			ServiceName: "book-server",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "book-server",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "book-server",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "book-server",
							Image:           "almasood/book-server:latest",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 3000,
								},
							},
							Args: []string{
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
	}

	fmt.Println("Creating StatefulSet...")
	result, err := statefulSetsClient.Create(context.TODO(), statefulSet, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Created StatefulSet %q.\n", result.GetObjectMeta().GetName())
	fmt.Println()
}

func updateStatefulSet(statefulSetsClient v1.StatefulSetInterface) {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := statefulSetsClient.Get(context.TODO(), "book-server", metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("failed to get latest version of StatefulSet: %v", getErr))
		}

		result.Spec.Replicas = ptr.To[int32](1)

		_, updateErr := statefulSetsClient.Update(context.TODO(), result, metav1.UpdateOptions{})

		return updateErr
	})

	if retryErr != nil {
		panic(fmt.Errorf("update failed: %v", retryErr))
	}
}

func deleteStatefulSet(statefulSetsClient v1.StatefulSetInterface) {
	deletePolicy := metav1.DeletePropagationForeground

	if err := statefulSetsClient.Delete(context.TODO(), "book-server", metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
}

func prompt() {
	fmt.Printf("-> Press return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	fmt.Println()
}
