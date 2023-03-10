package conthandler

import (
	"os"
	"path"
	"sniffer/pkg/config"
	configV1 "sniffer/pkg/config/v1"
	conthadlerV1 "sniffer/pkg/conthandler/v1"
	"sniffer/pkg/utils"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
)

var watcher *watch.FakeWatcher

type k8sFakeClient struct {
	Clientset *fake.Clientset
}

func (client *k8sFakeClient) GetApiVersion(workload any) string {
	return "v1"
}

func (client *k8sFakeClient) CalculateWorkloadParentRecursive(workload any) (string, string, error) {
	return "deployment", "nginx", nil
}

func (client *k8sFakeClient) GetWorkload(namespace, kind, name string) (any, error) {
	return "", nil
}

func (client *k8sFakeClient) GenerateWLID(workload any, clusterName string) string {
	return "wlid://cluster-" + clusterName + "/namespace-any" + "/deployment-nginx"
}

func (client *k8sFakeClient) GetWatcher() (watch.Interface, error) {
	watcher = watch.NewFake()
	return watcher, nil
}

func TestContWatcher(t *testing.T) {
	configPath := path.Join(utils.CurrentDir(), "..", "..", "configuration", "ConfigurationFile.json")
	err := os.Setenv(config.ConfigEnvVar, configPath)
	if err != nil {
		t.Fatalf("failed to set env ConfigEnvVar with err %v", err)
	}

	cfg := config.GetConfigurationConfigContext()
	configData, err := cfg.GetConfigurationReader()
	if err != nil {
		t.Fatalf("GetConfigurationReader failed with err %v", err)
	}
	err = cfg.ParseConfiguration(configV1.CreateFalcoMockConfigData(), configData)
	if err != nil {
		t.Fatalf("ParseConfiguration failed with err %v", err)
	}

	client := &k8sFakeClient{}
	client.Clientset = fake.NewSimpleClientset()
	contWatcher, err := CreateContainerWatcher(client)
	if err != nil {
		t.Fatalf("CreateContainerWatcher failed with error %v", err)
	}

	containersEventChan := make(chan conthadlerV1.ContainerEventData, 50)
	go func() {
		go contWatcher.StartWatchedOnContainers(containersEventChan)
		time.Sleep(1 * time.Second)
		go func() {
			pod := &v1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "nginx",
							Image:           "nginx",
							ImagePullPolicy: "Always",
						},
					},
				},
			}
			watcher.Add(pod)
			var cs []v1.ContainerStatus
			cs = make([]v1.ContainerStatus, 0)
			Started := true
			cs = append(cs, v1.ContainerStatus{
				Image:       "nginx",
				ContainerID: "nginxContainerID",
				Ready:       true,
				Started:     &Started,
			})
			pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, cs...)
			watcher.Modify(pod)
		}()
	}()

	event := <-containersEventChan
	if event.GetContainerEventType() != conthadlerV1.ContainerRunning {
		t.Fatalf("event container type is wrong, get: %s expected: %s", event.GetContainerEventType(), conthadlerV1.ContainerRunning)
	}
	if event.GetContainerID() != "nginxContainerID" {
		t.Fatalf("container ID is wrong,  get: %s expected: %s", event.GetContainerID(), "nginxContainerID")
	}
}
