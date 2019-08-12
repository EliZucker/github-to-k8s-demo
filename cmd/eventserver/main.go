package main

import (
	"fmt"
	"net/http"
	"gopkg.in/go-playground/webhooks.v5/github"
	"log"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	path = "/trigger"
)

func main() {
	// Get clientset:
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Github hook object w/ specified secret:
	hook, _ := github.New(github.Options.Secret("MyInsecureGitHubSecret"))

	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		log.Printf("event receieved")
		payload, err := hook.Parse(r, github.PushEvent, github.IssueCommentEvent)
		if err != nil {
			if err == github.ErrEventNotFound {
				// event wasn't one that we wanted to detect
			}
		}
		switch payload.(type) {

		// Make sample deployments on issue comments and pushes:
		case github.IssueCommentPayload:
			issueComment := payload.(github.IssueCommentPayload)
			makeBasicDeployment(*clientset, issueComment.Comment.Body)

		case github.PushPayload:
			push := payload.(github.PushPayload)
			makeBasicDeployment(*clientset, push.HeadCommit.Message)
		}
	})
	log.Printf("listening for github events!")
	http.ListenAndServe(":3000", nil)
}

func makeBasicDeployment(clientset kubernetes.Clientset, message string) {
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "alpine",
							Image: "alpine:3.10",
							Command: []string{"/bin/sh"},
							Args: []string{"-c", "echo " + message + "; sleep 999999"},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	fmt.Println("Creating deployment...")
	result, err := deploymentsClient.Create(deployment)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())
}

func int32Ptr(i int32) *int32 { return &i }