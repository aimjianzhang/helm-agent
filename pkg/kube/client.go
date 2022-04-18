package kube

import (
	context2 "context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"helm-example/pkg/model"
	"io"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
	"net/http"
	"strings"
)

func NewClient(f cmdutil.Factory) (*kubernetes.Clientset, error) {
	kubeClient, err := f.KubernetesClientSet()
	if err != nil {
		return nil, fmt.Errorf("get kubernetes client: %v", err)
	}
	return kubeClient, nil
}

func GetDeploymentInfo(context *model.Context, cmd *model.Packet) ([]*model.Packet, *model.Packet){
	var deploymentParams model.DeploymentParams
	err := json.Unmarshal([]byte(cmd.Payload), &deploymentParams)
	if err != nil{
		glog.Errorf("json unmarshal error: %s", err.Error())
		return nil,  &model.Packet{
			Type: model.DeploymentInfo,
			Payload: err.Error(),
		}
	}
	deployment, _ := context.Client.AppsV1().Deployments(deploymentParams.Namespace).Get(context2.TODO(), deploymentParams.DeploymentName, meta_v1.GetOptions{})
	deploymentResource, err := json.Marshal(deployment)
	if err != nil {
		return nil, &model.Packet{
			Type: model.DeploymentInfo,
			Payload: err.Error(),
		}
	}
	return nil, &model.Packet{
		Type: model.DeploymentInfo,
		Payload: string(deploymentResource),
	}
}

// ContainerExeCmd 容器中执行命令
func ContainerExeCmd(context *model.Context, request *http.Request, conn *websocket.Conn) {

	//升级get请求为webSocket协议
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	local := struct {
		io.Reader
		io.Writer
	}{
		r1, w2,
	}
	remote := struct {
		io.Reader
		io.Writer
	}{
		r2, w1,
	}

	go func() {
		exeCmd(context, request, local)
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := remote.Read(buf)
			if err != nil {
				return
			}
			glog.Info(string(buf[:n]))
			//写入ws数据
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	for {
		//读取ws中的数据
		_, message, err := conn.ReadMessage()

		if err != nil {
			return
		}
		if _, err := remote.Write(message); err != nil {
			return
		}
	}
}

func exeCmd(context *model.Context, request *http.Request, local io.ReadWriter) {
	paramMap := make(map[string]string)
	queryParams := request.URL.RawQuery
	queryParamsPart := strings.Split(queryParams, "&")
	for i := range queryParamsPart {
		queryParamKeyValue := strings.Split(queryParamsPart[i], "=")
		paramMap[queryParamKeyValue[0]] = queryParamKeyValue[1]
	}
	namespace := paramMap["namespace"]
	podName := paramMap["podName"]
	containerName := paramMap["containerName"]

	config, err := context.CmdFactory.ToRESTConfig()
	if err != nil {
		return
	}
	pod, err := context.Client.CoreV1().Pods(namespace).Get(context2.TODO() , podName, meta_v1.GetOptions{})

	if err != nil {
		glog.Errorf("can not find pod %s :%v", "hzero-register-7cf8458dd7-stq8c", err)
	}

	if pod.Status.Phase == core_v1.PodSucceeded || pod.Status.Phase == core_v1.PodFailed {
		return
	}
	validShells := []string{"bash", "sh", "powershell", "cmd"}
	for _, testShell := range validShells {
		cmd := []string{testShell}
		req := context.Client.CoreV1().RESTClient().Post().
			Resource("pods").
			Name(podName).
			Namespace(namespace).SubResource("exec").
			Param("container", containerName)
		req.VersionedParams(&core_v1.PodExecOptions{
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
			Container: containerName,
			Command:   cmd,
		}, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(config, http.MethodPost, req.URL())
		if err == nil {
			err = exec.Stream(remotecommand.StreamOptions{
				Stdin:  local,
				Stdout: local,
				Stderr: local,
				Tty:    true,
			})

			if err == nil {
				return
			}
		}
	}
	glog.Errorf("no support command")
}
