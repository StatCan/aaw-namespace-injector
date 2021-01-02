package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func cleanName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

func mutate(request v1beta1.AdmissionRequest) (v1beta1.AdmissionResponse, error) {
	response := v1beta1.AdmissionResponse{}

	// Default response
	response.Allowed = true
	response.UID = request.UID

	// Decode the pod object
	var err error
	pod := v1.Pod{}
	if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
		return response, fmt.Errorf("unable to decode Pod %w", err)
	}

	log.Printf("Check pod for notebook name %s/%s", pod.Namespace, pod.Name)

	// Only inject custom environment variables into notebook pods (condition: has notebook-name label)
	isNotebook := false
	if _, ok := pod.ObjectMeta.Labels["notebook-name"]; ok {
		isNotebook = true
	}

	if isNotebook {
		log.Printf("Found notebook name for %s/%s", pod.Namespace, pod.Name)

		patch := v1beta1.PatchTypeJSONPatch
		response.PatchType = &patch

		response.AuditAnnotations = map[string]string{
			"namespace-admission-controller": "Added custom environment variables",
		}

		patches := []map[string]interface{}{
			{
				"op":   "add",
				"path": "/spec/containers/0/env/-",
				"value": v1.EnvVar{
					Name:  "NB_NAMESPACE",
					Value: pod.Namespace,
				},
			},
		}

		response.Patch, err = json.Marshal(patches)
		if err != nil {
			return response, err
		}

		response.Result = &metav1.Status{
			Status: metav1.StatusSuccess,
		}
	} else {
		log.Printf("Notebook name not found for %s/%s", pod.Namespace, pod.Name)
	}

	return response, nil
}
