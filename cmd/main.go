package main

import (
	context2 "context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"helm-example/pkg/command"
	"helm-example/pkg/kube"
	"helm-example/pkg/model"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"net/http"
	"os"
)

var Context *model.Context

//设置websocket
//CheckOrigin防止跨站点的请求伪造
var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	getter := genericclioptions.NewConfigFlags(true)
	cmdFactory := cmdutil.NewFactory(getter)
	// 获得集群版本
	discoveryClient, err := cmdFactory.ToDiscoveryClient()
	if err != nil {
		return
	}
	version, err := discoveryClient.ServerVersion()
	if err != nil {
		return
	}
	glog.Info("version %s", version)
	client, err := kube.NewClient(cmdFactory)
	if err != nil {
		glog.Info(err.Error())
	}
	glog.Info("get kube client success")
	checkKube(client)
	channel := model.NewChannel(100, 100)
	Context = &model.Context{
		Client:     client,
		CmdFactory: cmdFactory,
		Channel:    channel,
	}
	go cmdManager()
	r := gin.Default()
	r.GET("/exe-cmd", exeCmd)
	r.GET("/container/exe-cmd", containerExeCmd)
	r.Run(":8089")
}

func containerExeCmd(c *gin.Context) {
	//升级get请求为webSocket协议
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
	}
	defer ws.Close() //返回前关闭

	kube.ContainerExeCmd(Context, c.Request, ws)
}

func cmdManager() {
	for {
		select {
		case cmd := <-Context.Channel.CommandChannel:
			go func(cmd *model.Packet) {
				if cmd == nil {
					glog.Error("got wrong command")
					return
				}
				var newCmds []*model.Packet = nil
				var resp *model.Packet = nil
				if processCmdFunc, ok := command.Funcs[cmd.Type]; ok {
					newCmds, resp = processCmdFunc(Context, cmd)
				} else {
					err := fmt.Errorf("type %s not exist", cmd.Type)
					glog.Info(err.Error())
				}
				if newCmds != nil {
					go func(newCmds []*model.Packet) {
						for i := 0; i < len(newCmds); i++ {
							Context.Channel.CommandChannel <- newCmds[i]
						}
					}(newCmds)
				}
				if resp != nil {
					go func(resp *model.Packet) {
						Context.Channel.ResponseChannel <- resp
					}(resp)
				}
			}(cmd)
		}
	}
}

//websocket实现
func exeCmd(c *gin.Context) {
	newWebsocket(c)
}

func newWebsocket(c *gin.Context) {
	//升级get请求为webSocket协议
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
	}
	defer ws.Close() //返回前关闭

	go func() {
		select {
		case resp, ok := <-Context.Channel.ResponseChannel:
			if !ok {
				return
			}
			//写入ws数据
			if err := ws.WriteJSON(resp); err != nil {
				return
			}
		}
	}()

	for {
		//读取ws中的数据
		packet := &model.Packet{}
		err := ws.ReadJSON(packet)
		if err != nil {
			return
		}
		Context.Channel.CommandChannel <- packet
	}
}

func checkKube(client *kubernetes.Clientset) {
	_, err := client.CoreV1().Pods("").List(context2.TODO(), meta_v1.ListOptions{})

	if err != nil {
		os.Exit(0)
	}

}
