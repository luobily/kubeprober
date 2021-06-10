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

type Config struct {
	Debug                   bool   `default:"false" desc:"enable debug logging"`
	CollectClusterInfo      bool   `default:"true" desc:"enable collect cluster info"`
	ClusterDialEndpoint     string `desc:"cluster dialer endpoint"`
	ClusterHeatBeatEndpoint string `desc:"cluster heartbeat endpoint"`
	ClusterKey              string `desc:"cluster key"`
	SecretKey               string `desc:"secret key"`
	K8SApiServerAddr        string `desc:"kube-apiserver address in cluster"`
}