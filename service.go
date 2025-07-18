package main

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"path/filepath"
)

func main() {
	kubeconfig := filepath.Join("/home", "abdullah", ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error loading kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset %v", err)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "book-server-service",
		},

		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "book-server",
			},
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Port:       3000,
					TargetPort: intstrPtr(3000),
					NodePort:   30080,
				},
			},
		},
	}

	svcClient := clientset.CoreV1().Services("default")
	result, err := svcClient.Create(context.TODO(), service, metav1.CreateOptions{})

	if err != nil {
		log.Fatalf("Failed to create service %v", err)
	}

	log.Printf("Service %q created.\n", result.GetName())
}

func intstrPtr(i int) intstr.IntOrString {
	return intstr.FromInt(i)
}
