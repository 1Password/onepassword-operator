package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const (
	// binVolumeName is the name of the volume where the OP CLI binary is stored.
	binVolumeName = "op-bin"

	// binVolumeMountPath is the mount path where the OP CLI binary can be found.
	binVolumeMountPath = "/op/bin/"

	connectTokenEnv = "OP_CONNECT_TOKEN"
	connectHostEnv  = "OP_CONNECT_HOST"
)

// binVolume is the shared, in-memory volume where the OP CLI binary lives.
var binVolume = corev1.Volume{
	Name: binVolumeName,
	VolumeSource: corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{
			Medium: corev1.StorageMediumMemory,
		},
	},
}

// binVolumeMount is the shared volume mount where the OP CLI binary lives.
var binVolumeMount = corev1.VolumeMount{
	Name:      binVolumeName,
	MountPath: binVolumeMountPath,
	ReadOnly:  true,
}

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

const (
	injectionStatus   = "operator.1password.io/status"
	injectAnnotation  = "operator.1password.io/inject"
	versionAnnotation = "operator.1password.io/version"
)

type SecretInjector struct {
	Config Config
	Server *http.Server
}

// the command line parameters for configuraing the webhook
type SecretInjectorParameters struct {
	Port     int    // webhook server port
	CertFile string // path to the x509 certificate for https
	KeyFile  string // path to the x509 private key matching `CertFile`
}

type Config struct {
	ConnectHost      string // the host in which a connect server is running
	ConnectTokenName string // the token name of the secret that stores the connect token
	ConnectTokenKey  string // the name of the data field in the secret the stores the connect token
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	_ = v1.AddToScheme(runtimeScheme)
}

func applyDefaultsWorkaround(containers []corev1.Container, volumes []corev1.Volume) {
	defaulter.Default(&corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: containers,
			Volumes:    volumes,
		},
	})
}

// Check if the pod should have secrets injected
func mutationRequired(metadata *metav1.ObjectMeta) bool {

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	status := annotations[injectionStatus]
	_, enabled := annotations[injectAnnotation]

	// if pod has not already been injected and injection has been enabled mark the pod for injection
	required := false
	if strings.ToLower(status) != "injected" && enabled {
		required = true
	}

	glog.Infof("Pod %v at namepspace %v. Secret injection status: %v Secret Injection Enabled:%v", metadata.Name, metadata.Namespace, status, required)
	return required
}

func addContainers(target, added []corev1.Container, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func addVolume(target, added []corev1.Volume, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Volume{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, patchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, patchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}

// mutation process for injecting secrets into pods
func (s *SecretInjector) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	ctx := context.Background()
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("Checking if secret injection is needed for %v %s at namespace %v",
		req.Kind, pod.Name, req.Namespace)

	// determine whether to inject secrets
	if !mutationRequired(&pod.ObjectMeta) {
		glog.Infof("Secret injection not required for %s at namespace %s", pod.Name, pod.Namespace)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	containersStr := pod.Annotations[injectAnnotation]

	containers := map[string]struct{}{}

	if containersStr == "" {
		glog.Infof("No containers set for secret injection for %s/%s", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	for _, container := range strings.Split(containersStr, ",") {
		containers[container] = struct{}{}
	}

	version, ok := pod.Annotations[versionAnnotation]
	if !ok {
		version = "2.0.0-beta.4"
	}

	mutated := false

	var patch []patchOperation
	for i, c := range pod.Spec.InitContainers {
		_, mutate := containers[c.Name]
		if !mutate {
			continue
		}
		c, didMutate, initContainerPatch, err := s.mutateContainer(ctx, &c, i)
		if err != nil {
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		if didMutate {
			mutated = true
			pod.Spec.InitContainers[i] = *c
		}
		patch = append(patch, initContainerPatch...)
	}

	for i, c := range pod.Spec.Containers {
		_, mutate := containers[c.Name]
		if !mutate {
			continue
		}

		c, didMutate, containerPatch, err := s.mutateContainer(ctx, &c, i)
		if err != nil {
			glog.Error("Error occured mutating container for secret injection: ", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		patch = append(patch, containerPatch...)
		if didMutate {
			mutated = true
			pod.Spec.Containers[i] = *c
		}
	}

	if !mutated {
		glog.Infof("No containers set for secret injection for %s/%s", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	// binInitContainer is the container that pulls the OP CLI
	// into a shared volume mount.
	var binInitContainer = corev1.Container{
		Name:            "copy-op-bin",
		Image:           "1password/op" + ":" + version,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{"sh", "-c",
			fmt.Sprintf("cp /usr/local/bin/op %s", binVolumeMountPath)},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      binVolumeName,
				MountPath: binVolumeMountPath,
			},
		},
	}

	patchBytes, err := createOPCLIPatch(&pod, []corev1.Container{binInitContainer}, patch)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// create mutation patch for resoures
func createOPCLIPatch(pod *corev1.Pod, containers []corev1.Container, patch []patchOperation) ([]byte, error) {

	annotations := map[string]string{injectionStatus: "injected"}
	patch = append(patch, addVolume(pod.Spec.Volumes, []corev1.Volume{binVolume}, "/spec/volumes")...)
	patch = append(patch, addContainers(pod.Spec.InitContainers, containers, "/spec/initContainers")...)
	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)

	return json.Marshal(patch)
}

func createOPConnectPatch(container *corev1.Container, containerIndex int, host, tokenSecretName, tokenSecretKey string) []patchOperation {
	var patch []patchOperation
	envs := []corev1.EnvVar{}

	// if connect configuration is already set in the container do not overwrite it
	hostConfig, tokenConfig := isConnectConfigurationSet(container)

	if !hostConfig {
		connectHostEnvVar := corev1.EnvVar{
			Name:  "OP_CONNECT_HOST",
			Value: host,
		}
		envs = append(envs, connectHostEnvVar)
	}

	if !tokenConfig {
		connectTokenEnvVar := corev1.EnvVar{
			Name: "OP_CONNECT_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: tokenSecretKey,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tokenSecretName,
					},
				},
			},
		}
		envs = append(envs, connectTokenEnvVar)
	}

	patch = append(patch, setEnvironment(*container, containerIndex, envs, "/spec/containers")...)

	return patch
}

func isConnectConfigurationSet(container *corev1.Container) (bool, bool) {

	hostConfig := false
	tokenConfig := false

	for _, env := range container.Env {
		if env.Name == connectHostEnv {
			hostConfig = true
		}

		if env.Name == connectTokenEnv {
			tokenConfig = true
		}

		if tokenConfig && hostConfig {
			break
		}
	}
	return hostConfig, tokenConfig
}

// mutates the container to allow for secrets to be injected into the container via the op cli
func (s *SecretInjector) mutateContainer(_ context.Context, container *corev1.Container, containerIndex int) (*corev1.Container, bool, []patchOperation, error) {
	//  prepending op run command to the container command so that secrets are injected before the main process is started
	if len(container.Command) == 0 {
		return container, false, nil, fmt.Errorf("not attaching OP to the container %s: the podspec does not define a command", container.Name)
	}

	// Prepend the command with op run --
	container.Command = append([]string{binVolumeMountPath + "op", "run", "--"}, container.Command...)

	var patch []patchOperation

	// adding the cli to the container using a volume mount
	path := fmt.Sprintf("%s/%d/volumeMounts", "/spec/containers", containerIndex)
	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  path,
		Value: []corev1.VolumeMount{binVolumeMount},
	})

	// replacing the container command with a command prepended with op run
	path = fmt.Sprintf("%s/%d/command", "/spec/containers", containerIndex)
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  path,
		Value: container.Command,
	})

	//creating patch for adding connect environment variables to container. If they are already set in the container then this will be skipped
	patch = append(patch, createOPConnectPatch(container, containerIndex, s.Config.ConnectHost, s.Config.ConnectTokenName, s.Config.ConnectTokenKey)...)
	return container, true, patch, nil
}

func setEnvironment(container corev1.Container, containerIndex int, addedEnv []corev1.EnvVar, basePath string) (patch []patchOperation) {
	first := len(container.Env) == 0
	var value interface{}
	for _, add := range addedEnv {
		path := fmt.Sprintf("%s/%d/env", basePath, containerIndex)
		value = add
		if first {
			first = false
			value = []corev1.EnvVar{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

// Serve method for secrets injector webhook
func (s *SecretInjector) Serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = s.mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
