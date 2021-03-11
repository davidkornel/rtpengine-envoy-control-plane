// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
package controlplane

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/golang/protobuf/ptypes"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	udp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/udp/udp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
)

var (
	ingressVersion   = 0
	sidecarVersion   = 0
	ingressListeners = make([]udpListener, 0)
	sidecarListeners = make([]udpListener, 0)
)

type udpListener struct {
	listenerName string
	cluster      udpCluster
	//socket listenerPort where udpListener listens for incoming datagrams
	listenerPort uint32
	//socket listenerAddress where udpListener listens for incoming datagrams
	listenerAddress string
}

type udpCluster struct {
	clusterName string
	//upstreamPort is the listenerPort of an endpoint where udpCluster forwards the datagrams
	upstreamPort uint32
	//upstreamHost is the listenerAddress of an endpoint where udpCluster forwards the datagrams
	upstreamHost string
}

func makeUDPClusters(pod string) *[]types.Resource {
	var clusters []types.Resource
	if pod == "ingress" {
		for _, list := range ingressListeners {
			clusters = append(clusters, *makeUDPCluster(list.cluster)...)
		}
	} else if pod == "sidecar" {
		for _, list := range sidecarListeners {
			clusters = append(clusters, *makeUDPCluster(list.cluster)...)
		}
	}
	return &clusters
}

func makeUDPCluster(udpClust udpCluster) *[]types.Resource {

	clust := &cluster.Cluster{
		Name:           udpClust.clusterName,
		ConnectTimeout: ptypes.DurationProto(1 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{
			Type: cluster.Cluster_STRICT_DNS,
		},
		LbPolicy:       cluster.Cluster_ROUND_ROBIN,
		LoadAssignment: makeEndpoint(udpClust),
	}
	return &[]types.Resource{clust}
}

func makeEndpoint(udpClust udpCluster) *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: udpClust.clusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_UDP,
									Address:  udpClust.upstreamHost,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: udpClust.upstreamPort,
									},
								},
							},
						},
					},
				},
			}},
		}},
	}
}

func makeUDPListeners(pod string) *[]types.Resource {
	var listeners []types.Resource
	if pod == "ingress" {
		for _, list := range ingressListeners {
			listeners = append(listeners, *makeUDPListener(list)...)
		}
	} else if pod == "sidecar" {
		for _, list := range sidecarListeners {
			listeners = append(listeners, *makeUDPListener(list)...)
		}
	}
	return &listeners
}

func makeUDPListener(udpList udpListener) *[]types.Resource {

	udpFilter := &udp.UdpProxyConfig{
		StatPrefix: udpList.listenerName,
		RouteSpecifier: &udp.UdpProxyConfig_Cluster{
			Cluster: udpList.cluster.clusterName,
		},
	}

	pbst, err := ptypes.MarshalAny(udpFilter)
	if err != nil {
		panic(err)
	}

	list := &listener.Listener{
		Name:      udpList.listenerName,
		ReusePort: true,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_UDP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: udpList.listenerPort,
					},
				},
			},
		},
		ListenerFilters: []*listener.ListenerFilter{{
			Name: "envoy.filters.udp_listener.udp_proxy",
			ConfigType: &listener.ListenerFilter_TypedConfig{
				TypedConfig: pbst,
			},
		}},
	}
	return &[]types.Resource{list}
}

func GenerateSnapshot(pod string) cachev3.Snapshot {
	if pod == "ingress" {
		ingressVersion++
	} else if pod == "sidecar" {
		sidecarVersion++
	} else {
		log.Fatal("Envoy NodeId not known: ", pod)
	}
	return cachev3.NewSnapshot(
		strconv.Itoa(ingressVersion),
		[]types.Resource{}, // endpoints
		*makeUDPClusters(pod),
		[]types.Resource{},     //routes
		*makeUDPListeners(pod), //listeners
		[]types.Resource{},     // runtimes
		[]types.Resource{},     // secrets
	)
}

func createNewListeners(m Message) {
	ingressListeners = make([]udpListener, 0)
	sidecarListeners = make([]udpListener, 0)
	//INGRESS
	var rtpAI = udpListener{
		listenerName: fmt.Sprintf("ingress-l-rtp-a-%d", ingressVersion+1),
		cluster: udpCluster{
			clusterName:  fmt.Sprintf("ingress-c-rtp-a-%d", ingressVersion+1),
			upstreamPort: 19000,
			upstreamHost: "worker.default.svc",
		},
		listenerPort:    m.CallerRTP,
		listenerAddress: "0.0.0.0",
	}
	var rtcpAI = udpListener{
		listenerName: fmt.Sprintf("ingress-l-rtcp-a-%d", ingressVersion+1),
		cluster: udpCluster{
			clusterName:  fmt.Sprintf("ingress-c-rtcp-a-%d", ingressVersion+1),
			upstreamPort: 19001,
			upstreamHost: "worker.default.svc",
		},
		listenerPort:    m.CallerRTCP,
		listenerAddress: "0.0.0.0",
	}
	var rtpBI = udpListener{
		listenerName: fmt.Sprintf("ingress-l-rtp-b-%d", ingressVersion+1),
		cluster: udpCluster{
			clusterName:  fmt.Sprintf("ingress-c-rtp-b-%d", ingressVersion+1),
			upstreamPort: 19002,
			upstreamHost: "worker.default.svc",
		},
		listenerPort:    m.CalleeRTP,
		listenerAddress: "0.0.0.0",
	}
	var rtcpBI = udpListener{
		listenerName: fmt.Sprintf("ingress-l-rtcp-b-%d", ingressVersion+1),
		cluster: udpCluster{
			clusterName:  fmt.Sprintf("ingress-c-rtcp-b-%d", ingressVersion+1),
			upstreamPort: 19003,
			upstreamHost: "worker.default.svc",
		},
		listenerPort:    m.CalleeRTCP,
		listenerAddress: "0.0.0.0",
	}

	//WORKER/SIDECAR
	var rtpAW = udpListener{
		listenerName: fmt.Sprintf("worker-l-rtp-a-%d", sidecarVersion+1),
		cluster: udpCluster{
			clusterName:  fmt.Sprintf("worker-c-rtp-a-%d", sidecarVersion+1),
			upstreamPort: m.CallerRTP,
			upstreamHost: "127.0.0.1",
		},
		listenerPort:    19000,
		listenerAddress: "0.0.0.0",
	}
	var rtcpAW = udpListener{
		listenerName: fmt.Sprintf("worker-l-rtcp-a-%d", sidecarVersion+1),
		cluster: udpCluster{
			clusterName:  fmt.Sprintf("worker-c-rtcp-a-%d", sidecarVersion+1),
			upstreamPort: m.CallerRTCP,
			upstreamHost: "127.0.0.1",
		},
		listenerPort:    19001,
		listenerAddress: "0.0.0.0",
	}
	var rtpBW = udpListener{
		listenerName: fmt.Sprintf("worker-l-rtp-b-%d", sidecarVersion+1),
		cluster: udpCluster{
			clusterName:  fmt.Sprintf("worker-c-rtp-b-%d", sidecarVersion+1),
			upstreamPort: m.CalleeRTP,
			upstreamHost: "127.0.0.1",
		},
		listenerPort:    19002,
		listenerAddress: "0.0.0.0",
	}
	var rtcpBW = udpListener{
		listenerName: fmt.Sprintf("worker-l-rtcp-b-%d", sidecarVersion+1),
		cluster: udpCluster{
			clusterName:  fmt.Sprintf("worker-c-rtcp-b-%d", sidecarVersion+1),
			upstreamPort: m.CalleeRTCP,
			upstreamHost: "127.0.0.1",
		},
		listenerPort:    19003,
		listenerAddress: "0.0.0.0",
	}
	ingressListeners = append(ingressListeners, rtpAI, rtcpAI, rtpBI, rtcpBI)

	sidecarListeners = append(sidecarListeners, rtpAW, rtcpAW, rtpBW, rtcpBW)

}

func updateConfig(pod string, l *Logger) {

	// Create the snapshot that we'll serve to Envoy
	snapshot := GenerateSnapshot(pod)
	if err := snapshot.Consistent(); err != nil {
		l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
		os.Exit(1)
	}

	//// Clear the snapshot from the cache
	//// if there is an available snapshot for the given node
	//if snap, err := cache.GetSnapshot(pod); err == nil {
	//	l.Debugf("Snapshot to be cleared: %+v \n", snap)
	//	res := snap.GetResources("type.googleapis.com/envoy.config.listener.v3.Listener")
	//	fmt.Printf("\n\n\n")
	//	for key, value := range res {
	//		fmt.Println("Key:", key, "Value:", value)
	//		value.Reset()
	//	}
	//	fmt.Printf("\n\n\n")
	//} else {
	//	l.Debugf("No snapshot to be cleared: %s", err)
	//}

	l.Debugf("will serve snapshot %+v", snapshot)
	// Add the snapshot to the cache
	if err := cache.SetSnapshot(pod, snapshot); err != nil {
		l.Errorf("snapshot error %q for %+v", err, snapshot)
		os.Exit(1)
	}

}
