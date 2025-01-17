// Copyright (c) 2021 Terminus, Inc.
//
// This program is free software: you can use, redistribute, and/or modify
// it under the terms of the GNU Affero General Public License, version 3
// or later ("AGPL"), as published by the Free Software Foundation.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/erda-project/kubeprober/apistructs"
	"github.com/rancher/remotedialer"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var connected = make(chan struct{})

const (
	dailEndPointSuffix      = "/clusteragent/connect"
	heartBeatEndPointSuffix = "/heartbeat"
)

func sendHeartBeat(heartBeatAddr string, clusterName string, secretKey string) error {
	ctx := context.Background()
	var rsp *http.Response
	var err error
	var clientset *kubernetes.Clientset
	var version *version.Info
	var nodes *v1.NodeList

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		userHomeDir = ""
	}
	kubeConfig := filepath.Join(userHomeDir, ".kube", "config")
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			klog.Errorf("[remote dialer agent] get kubernetes client config error: %+v\n", err)
			return err
		}
	}

	config.AcceptContentTypes = "application/json"
	if clientset, err = kubernetes.NewForConfig(config); err != nil {
		return err
	}
	if version, err = clientset.ServerVersion(); err != nil {
		return err
	}
	if nodes, err = clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); err != nil {
		return err
	}
	hbData := apistructs.HeartBeatReq{
		Name:           clusterName,
		SecretKey:      secretKey,
		Address:        config.Host,
		ProbeNamespace: os.Getenv("POD_NAMESPACE"),
		CaData:         base64.StdEncoding.EncodeToString(config.CAData),
		CertData:       base64.StdEncoding.EncodeToString(config.CertData),
		KeyData:        base64.StdEncoding.EncodeToString(config.KeyData),
		Token:          base64.StdEncoding.EncodeToString([]byte(config.BearerToken)),
		Version:        version.String(),
		NodeCount:      len(nodes.Items),
	}
	json_data, _ := json.Marshal(hbData)
	if rsp, err = http.Post(heartBeatAddr, "application/json", bytes.NewBuffer(json_data)); err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(rsp.Body)
	if rsp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}
	rsp.Body.Close()
	return nil
}
func Start(ctx context.Context, cfg *Config) {
	var clusterDialEndpoint string
	var clusterHeartBeatEndpoint string
	headers := http.Header{
		"X-Cluster-Name": {cfg.ClusterName},
		"Secret-Key":     {cfg.SecretKey},
	}

	u, err := url.Parse(cfg.ProbeMasterAddr)
	if err != nil {
		klog.Errorf("[tunnel-client] get probe-master addr error: %+v\n", err)
		return
	}
	switch u.Scheme {
	case "http":
		clusterDialEndpoint = "ws://" + u.Host + dailEndPointSuffix
		clusterHeartBeatEndpoint = "http://" + u.Host + heartBeatEndPointSuffix
	case "https":
		clusterDialEndpoint = "wss://" + u.Host + dailEndPointSuffix
		clusterHeartBeatEndpoint = "https://" + u.Host + heartBeatEndPointSuffix
	}

	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				if err := sendHeartBeat(clusterHeartBeatEndpoint, cfg.ClusterName, cfg.SecretKey); err != nil {
					klog.Errorf("[heartbeat] send heartbeat request error: %+v\n", err)
					break
				}
			}
		}
	}()
	for {
		remotedialer.ClientConnect(ctx, clusterDialEndpoint, headers, nil, func(proto, address string) bool {
			switch proto {
			case "tcp":
				return true
			case "unix":
				return address == "/var/run/docker.sock"
			case "npipe":
				return address == "//./pipe/docker_engine"
			}
			return false
		}, onConnect)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(rand.Int()%10) * time.Second):
			// retry connect after sleep a random time
		}
	}

}

func onConnect(ctx context.Context, session *remotedialer.Session) error {
	connected <- struct{}{}
	return nil

}
