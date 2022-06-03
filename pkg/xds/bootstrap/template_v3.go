package bootstrap

import (
	"net"
	"strconv"

	"github.com/asaskevich/govalidator"
	envoy_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	envoy_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_grpc_credentials_v3 "github.com/envoyproxy/go-control-plane/envoy/config/grpc_credential/v3"
	envoy_metrics_v3 "github.com/envoyproxy/go-control-plane/envoy/config/metrics/v3"
	envoy_tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_type_matcher_v3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"

	util_proto "github.com/kumahq/kuma/pkg/util/proto"
	"github.com/kumahq/kuma/pkg/xds/envoy/tls"
)

func genConfig(parameters configParameters) (*envoy_bootstrap_v3.Bootstrap, error) {
	test := envoy_grpc_credentials_v3.FileBasedMetadataConfig{
		SecretData: &envoy_core_v3.DataSource{
			Specifier: &envoy_core_v3.DataSource_Filename{Filename: "/tmp/test-token"},
		},
	}

	res := &envoy_bootstrap_v3.Bootstrap{
		Node: &envoy_core_v3.Node{
			Id:      parameters.Id,
			Cluster: parameters.Service,
			Metadata: util_proto.MustStruct(map[string]interface{}{
				"version": map[string]interface{}{
					"kumaDp": map[string]interface{}{
						"version":   parameters.KumaDpVersion,
						"gitTag":    parameters.KumaDpGitTag,
						"gitCommit": parameters.KumaDpGitCommit,
						"buildDate": parameters.KumaDpBuildDate,
					},
					"envoy": map[string]interface{}{
						"version":          parameters.EnvoyVersion,
						"build":            parameters.EnvoyBuild,
						"kumaDpCompatible": parameters.EnvoyKumaDpCompatible,
					},
					"dependencies": map[string]interface{}{},
				},
			}),
		},
		LayeredRuntime: &envoy_bootstrap_v3.LayeredRuntime{
			Layers: []*envoy_bootstrap_v3.RuntimeLayer{
				{
					Name: "kuma",
					LayerSpecifier: &envoy_bootstrap_v3.RuntimeLayer_StaticLayer{
						StaticLayer: util_proto.MustStruct(map[string]interface{}{
							"envoy.restart_features.use_apple_api_for_dns_lookups": false,
							"re2.max_program_size.error_level":                     4294967295,
							"re2.max_program_size.warn_level":                      1000,
						}),
					},
				},
			},
		},
		StatsConfig: &envoy_metrics_v3.StatsConfig{
			StatsTags: []*envoy_metrics_v3.TagSpecifier{
				{
					TagName:  "name",
					TagValue: &envoy_metrics_v3.TagSpecifier_Regex{Regex: "^grpc\\.((.+)\\.)"},
				},
				{
					TagName:  "status",
					TagValue: &envoy_metrics_v3.TagSpecifier_Regex{Regex: "^grpc.*streams_closed(_([0-9]+))"},
				},
				{
					TagName:  "kafka_name",
					TagValue: &envoy_metrics_v3.TagSpecifier_Regex{Regex: "^kafka(\\.(\\S*[0-9]))\\."},
				},
				{
					TagName:  "kafka_type",
					TagValue: &envoy_metrics_v3.TagSpecifier_Regex{Regex: "^kafka\\..*\\.(.*)"},
				},
				{
					TagName:  "worker",
					TagValue: &envoy_metrics_v3.TagSpecifier_Regex{Regex: "(worker_([0-9]+)\\.)"},
				},
				{
					TagName:  "listener",
					TagValue: &envoy_metrics_v3.TagSpecifier_Regex{Regex: "((.+?)\\.)rbac\\."},
				},
			},
		},
		DynamicResources: &envoy_bootstrap_v3.Bootstrap_DynamicResources{
			LdsConfig: &envoy_core_v3.ConfigSource{
				ConfigSourceSpecifier: &envoy_core_v3.ConfigSource_Ads{Ads: &envoy_core_v3.AggregatedConfigSource{}},
				ResourceApiVersion:    envoy_core_v3.ApiVersion_V3,
			},
			CdsConfig: &envoy_core_v3.ConfigSource{
				ConfigSourceSpecifier: &envoy_core_v3.ConfigSource_Ads{Ads: &envoy_core_v3.AggregatedConfigSource{}},
				ResourceApiVersion:    envoy_core_v3.ApiVersion_V3,
			},
			AdsConfig: &envoy_core_v3.ApiConfigSource{
				ApiType:                   envoy_core_v3.ApiConfigSource_GRPC,
				TransportApiVersion:       envoy_core_v3.ApiVersion_V3,
				SetNodeOnFirstMessageOnly: true,
				GrpcServices: []*envoy_core_v3.GrpcService{
					// {
					// 	TargetSpecifier: &envoy_core_v3.GrpcService_EnvoyGrpc_{
					// 		EnvoyGrpc: &envoy_core_v3.GrpcService_EnvoyGrpc{
					// 			ClusterName: "ads_cluster",
					// 		},
					// 	},
					// },
					{
						TargetSpecifier: &envoy_core_v3.GrpcService_GoogleGrpc_{
							GoogleGrpc: &envoy_core_v3.GrpcService_GoogleGrpc{
								TargetUri:              "host.kuma-cp.system:5678",
								StatPrefix:             "ads",
								CredentialsFactoryName: "envoy.grpc_credentials.file_based_metadata",
								CallCredentials: []*envoy_core_v3.GrpcService_GoogleGrpc_CallCredentials{
									{
										CredentialSpecifier: &envoy_core_v3.GrpcService_GoogleGrpc_CallCredentials_FromPlugin{
											FromPlugin: &envoy_core_v3.GrpcService_GoogleGrpc_CallCredentials_MetadataCredentialsFromPlugin{
												Name: "envoy.grpc_credentials.file_based_metadata",
												ConfigType: &envoy_core_v3.GrpcService_GoogleGrpc_CallCredentials_MetadataCredentialsFromPlugin_TypedConfig{
													TypedConfig: util_proto.MustMarshalAny(&test),
												},
											},
										},
									},
								},
								ChannelCredentials: &envoy_core_v3.GrpcService_GoogleGrpc_ChannelCredentials{
									CredentialSpecifier: &envoy_core_v3.GrpcService_GoogleGrpc_ChannelCredentials_SslCredentials{
										SslCredentials: &envoy_core_v3.GrpcService_GoogleGrpc_SslCredentials{
											RootCerts: &envoy_core_v3.DataSource{
												Specifier: &envoy_core_v3.DataSource_InlineString{
													InlineString: `-----BEGIN CERTIFICATE-----
MIIDYTCCAkmgAwIBAgIRAKLYiwVtjQ8Qo27W/3udjs0wDQYJKoZIhvcNAQELBQAw
KTEnMCUGA1UEAxMea3VtYS1jb250cm9sLXBsYW5lLnRlc3Quc2VydmVyMB4XDTIy
MDYwMzE0MzkyMloXDTMyMDUzMTE0MzkyMlowKTEnMCUGA1UEAxMea3VtYS1jb250
cm9sLXBsYW5lLnRlc3Quc2VydmVyMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEA3CFIj9ribTMRoN3GXEzGT2Voc251InaYT7lXBjrnnlbr2SRp/Q8aoy2G
0ty/xc2kHtTjLOoxefwHUWHP9pV1POlajkn95N+OyMivkDrx517CFphao/6W4AWm
jDODGS7gT2af7PxHpiZdwjzhLzyf70PrLPzb6JtGjc8i2iXH8zxoJ0JztaLehpbc
O9BV5HCHAiRVv04ioNXTWPDLbARyDtoenkIsc6iGOppfMl+H8Cy7j6yCYi4sI6Wj
EYMWxgVcmjmtpHG2CmNvliuPTgwrImh7W6PLUj2msrFxZAhdIIKirHjqN5WNcqH9
Oxl4eAU4isnmKsDsigluG6vFh0sxnQIDAQABo4GDMIGAMA4GA1UdDwEB/wQEAwIC
pDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQW
BBRm+iZ0uUt2fRnVpFWOBaoMEVGZ0zApBgNVHREEIjAggh5rdW1hLWNvbnRyb2wt
cGxhbmUudGVzdC5zZXJ2ZXIwDQYJKoZIhvcNAQELBQADggEBAJmxN4mXqpjF5jjk
fJObWxPUjT/0upQNgH2Knvf8IyTC0QmNADkUg8rbSuXA5ELg/8AveF2GjaZhjZgt
vuQtP7HlXej8c0TLLmzIgooYntylCEqGUlBPg0nn/p8DsQNqcb+TfpqrVfZoyn7d
hea3cQzGB5TUlpRS9IU3o8ved31nz+D4B31CJQLboyofzkJmZeFAt6nZIX0juPKS
Qv1CeNkneR6JTXNwv9ZOa9TGl9yuXiP18uYkvqTEdyxYP5FXC9RNuxqfXBeY8Q31
aRhsYrU9EqpKDnwzvBIouiLFNM99TDQD3olEO2SSmnm+BN7zZYyQeM2CMw5AezuJ
hKqYKA4=
-----END CERTIFICATE-----`,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		// 			},
		// 		},
		// 	},
		// },

		// 		- googleGrpc:
		//         callCredentials:
		//         - fromPlugin:
		//             name: envoy.grpc_credentials.file_based_metadata
		//             typedConfig:
		//               '@type': type.googleapis.com/envoy.config.grpc_credential.v3.FileBasedMetadataConfig
		//               secretData:
		//                 filename: {{ .DataplaneTokenPath }}
		//         credentialsFactoryName: envoy.grpc_credentials.file_based_metadata
		// {{ if .CertBytes}}
		//         channelCredentials:
		//           sslCredentials:
		//             rootCerts:
		//               inlineBytes: {{ .CertBytes }}
		// {{ end }}
		// statPrefix: ads
		// targetUri: {{ .XdsHost }}:{{ .XdsPort }}
		StaticResources: &envoy_bootstrap_v3.Bootstrap_StaticResources{
			Secrets: []*envoy_tls.Secret{
				{
					Name: tls.CpValidationCtx,
					Type: &envoy_tls.Secret_ValidationContext{
						ValidationContext: &envoy_tls.CertificateValidationContext{
							MatchSubjectAltNames: []*envoy_type_matcher_v3.StringMatcher{
								{
									MatchPattern: &envoy_type_matcher_v3.StringMatcher_Exact{Exact: parameters.XdsHost},
								},
							},
							TrustedCa: &envoy_core_v3.DataSource{
								Specifier: &envoy_core_v3.DataSource_InlineBytes{
									InlineBytes: parameters.CertBytes,
								},
							},
						},
					},
				},
			},
			Clusters: []*envoy_cluster_v3.Cluster{
				{
					// TODO does timeout and keepAlive make sense on this as it uses unix domain sockets?
					Name:                 "access_log_sink",
					ConnectTimeout:       util_proto.Duration(parameters.XdsConnectTimeout),
					Http2ProtocolOptions: &envoy_core_v3.Http2ProtocolOptions{},
					LbPolicy:             envoy_cluster_v3.Cluster_ROUND_ROBIN,
					UpstreamConnectionOptions: &envoy_cluster_v3.UpstreamConnectionOptions{
						TcpKeepalive: &envoy_core_v3.TcpKeepalive{
							KeepaliveProbes:   util_proto.UInt32(3),
							KeepaliveTime:     util_proto.UInt32(10),
							KeepaliveInterval: util_proto.UInt32(10),
						},
					},
					ClusterDiscoveryType: &envoy_cluster_v3.Cluster_Type{Type: envoy_cluster_v3.Cluster_STATIC},
					LoadAssignment: &envoy_config_endpoint_v3.ClusterLoadAssignment{
						ClusterName: "access_log_sink",
						Endpoints: []*envoy_config_endpoint_v3.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoy_config_endpoint_v3.LbEndpoint{
									{
										HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
											Endpoint: &envoy_config_endpoint_v3.Endpoint{
												Address: &envoy_core_v3.Address{
													Address: &envoy_core_v3.Address_Pipe{Pipe: &envoy_core_v3.Pipe{Path: parameters.AccessLogPipe}},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Name:                 "ads_cluster",
					ConnectTimeout:       util_proto.Duration(parameters.XdsConnectTimeout),
					Http2ProtocolOptions: &envoy_core_v3.Http2ProtocolOptions{},
					LbPolicy:             envoy_cluster_v3.Cluster_ROUND_ROBIN,
					UpstreamConnectionOptions: &envoy_cluster_v3.UpstreamConnectionOptions{
						TcpKeepalive: &envoy_core_v3.TcpKeepalive{
							KeepaliveProbes:   util_proto.UInt32(3),
							KeepaliveTime:     util_proto.UInt32(10),
							KeepaliveInterval: util_proto.UInt32(10),
						},
					},
					ClusterDiscoveryType: &envoy_cluster_v3.Cluster_Type{Type: clusterTypeFromHost(parameters.XdsHost)},
					DnsLookupFamily:      dnsLookupFamilyFromXdsHost(parameters.XdsHost, net.LookupIP),
					LoadAssignment: &envoy_config_endpoint_v3.ClusterLoadAssignment{
						ClusterName: "ads_cluster",
						Endpoints: []*envoy_config_endpoint_v3.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoy_config_endpoint_v3.LbEndpoint{
									{
										HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
											Endpoint: &envoy_config_endpoint_v3.Endpoint{
												Address: &envoy_core_v3.Address{
													Address: &envoy_core_v3.Address_SocketAddress{
														SocketAddress: &envoy_core_v3.SocketAddress{
															Address:       parameters.XdsHost,
															PortSpecifier: &envoy_core_v3.SocketAddress_PortValue{PortValue: parameters.XdsPort},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, r := range res.StaticResources.Clusters {
		if r.Name == "ads_cluster" {
			transport := &envoy_tls.UpstreamTlsContext{
				Sni: parameters.XdsHost,
				CommonTlsContext: &envoy_tls.CommonTlsContext{
					TlsParams: &envoy_tls.TlsParameters{
						TlsMinimumProtocolVersion: envoy_tls.TlsParameters_TLSv1_2,
					},
					ValidationContextType: &envoy_tls.CommonTlsContext_ValidationContextSdsSecretConfig{
						ValidationContextSdsSecretConfig: &envoy_tls.SdsSecretConfig{
							Name: tls.CpValidationCtx,
						},
					},
				},
			}
			any, err := util_proto.MarshalAnyDeterministic(transport)
			if err != nil {
				return nil, err
			}
			r.TransportSocket = &envoy_core_v3.TransportSocket{
				Name: "envoy.transport_sockets.tls",
				ConfigType: &envoy_core_v3.TransportSocket_TypedConfig{
					TypedConfig: any,
				},
			}
		}
	}
	if parameters.HdsEnabled {
		res.HdsConfig = &envoy_core_v3.ApiConfigSource{
			ApiType:                   envoy_core_v3.ApiConfigSource_GRPC,
			TransportApiVersion:       envoy_core_v3.ApiVersion_V3,
			SetNodeOnFirstMessageOnly: true,
			GrpcServices: []*envoy_core_v3.GrpcService{
				{
					TargetSpecifier: &envoy_core_v3.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &envoy_core_v3.GrpcService_EnvoyGrpc{
							ClusterName: "ads_cluster",
						},
					},
				},
			},
		}
	}

	// if parameters.DataplaneToken != "" {
	// 	if res.HdsConfig != nil {
	// 		for _, n := range res.HdsConfig.GrpcServices {
	// 			n.InitialMetadata = []*envoy_core_v3.HeaderValue{
	// 				{Key: "authorization", Value: parameters.DataplaneToken},
	// 			}
	// 		}
	// 	}
	// 	for _, n := range res.DynamicResources.AdsConfig.GrpcServices {
	// 		n.InitialMetadata = []*envoy_core_v3.HeaderValue{
	// 			{Key: "authorization", Value: parameters.DataplaneToken},
	// 		}
	// 	}
	// }
	if parameters.DataplaneResource != "" {
		res.Node.Metadata.Fields["dataplane.resource"] = util_proto.MustNewValueForStruct(parameters.DataplaneResource)
	}
	if parameters.AdminPort != 0 {
		res.Node.Metadata.Fields["dataplane.admin.port"] = util_proto.MustNewValueForStruct(strconv.Itoa(int(parameters.AdminPort)))
		res.Admin = &envoy_bootstrap_v3.Admin{
			AccessLogPath: parameters.AdminAccessLogPath,
			Address: &envoy_core_v3.Address{
				Address: &envoy_core_v3.Address_SocketAddress{
					SocketAddress: &envoy_core_v3.SocketAddress{
						Address:  parameters.AdminAddress,
						Protocol: envoy_core_v3.SocketAddress_TCP,
						PortSpecifier: &envoy_core_v3.SocketAddress_PortValue{
							PortValue: parameters.AdminPort,
						},
					},
				},
			},
		}
	}
	if parameters.DNSPort != 0 {
		res.Node.Metadata.Fields["dataplane.dns.port"] = util_proto.MustNewValueForStruct(strconv.Itoa(int(parameters.DNSPort)))
	}
	if parameters.EmptyDNSPort != 0 {
		res.Node.Metadata.Fields["dataplane.dns.empty.port"] = util_proto.MustNewValueForStruct(strconv.Itoa(int(parameters.EmptyDNSPort)))
	}
	if parameters.ProxyType != "" {
		res.Node.Metadata.Fields["dataplane.proxyType"] = util_proto.MustNewValueForStruct(parameters.ProxyType)
	}
	if len(parameters.DynamicMetadata) > 0 {
		md := make(map[string]interface{}, len(parameters.DynamicMetadata))
		for k, v := range parameters.DynamicMetadata {
			md[k] = v
		}
		res.Node.Metadata.Fields["dynamicMetadata"] = util_proto.MustNewValueForStruct(md)
	}
	return res, nil
}

func dnsLookupFamilyFromXdsHost(host string, lookupFn func(host string) ([]net.IP, error)) envoy_cluster_v3.Cluster_DnsLookupFamily {
	if govalidator.IsDNSName(host) && host != "localhost" {
		ips, err := lookupFn(host)
		if err != nil {
			log.Info("[WARNING] error looking up XDS host to determine DnsLookupFamily, falling back to AUTO", "hostname", host)
			return envoy_cluster_v3.Cluster_AUTO
		}
		hasIPv6 := false

		for _, ip := range ips {
			if ip.To4() == nil {
				hasIPv6 = true
			}
		}

		if !hasIPv6 && len(ips) > 0 {
			return envoy_cluster_v3.Cluster_V4_ONLY
		}
	}

	return envoy_cluster_v3.Cluster_AUTO // default
}

func clusterTypeFromHost(host string) envoy_cluster_v3.Cluster_DiscoveryType {
	if govalidator.IsIP(host) {
		return envoy_cluster_v3.Cluster_STATIC
	}
	return envoy_cluster_v3.Cluster_STRICT_DNS
}
