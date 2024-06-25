module github.com/containerd/nerdctl/v2

go 1.22.0

toolchain go1.22.3

//replace github.com/containerd/containerd/v2 => github.com/containerd/containerd/v2 v2.0.0-rc.3

//replace github.com/containerd/containerd v1.7.0 => github.com/containerd/containerd/api v1.8.0-rc.2
//replace github.com/containerd/containerd v1.7.8 => github.com/containerd/containerd/api v1.8.0-rc.2
// replace github.com/containerd/containerd v1.7.18 => github.com/containerd/containerd/api v1.8.0-rc.2
// replace github.com/containerd/containerd/images/archive => github.com/containerd/containerd/v2/core/images/archive v2.0.0-rc.3

//replace github.com/containerd/containerd/api/types => github.com/containerd/containerd/api/types v1.8.0-rc.2
//replace github.com/containerd/containerd/api => github.com/containerd/containerd/api v1.8.0-rc.2
//replace github.com/containerd/containerd/api/types/transfer => github.com/containerd/containerd/api/types/transfer v1.8.0-rc.2
//replace github.com/containerd/containerd/api/services/transfer => github.com/containerd/containerd/api/services/transfer v1.8.0-rc.2
// replace github.com/containerd/stargz-snapshotter/ipfs/client => github.com/apostasie/stargz-snapshotter/ipfs/client v0.0.0-20240624071742-7014809d9d17
//replace github.com/containerd/containerd/api/events => github.com/containerd/containerd/api/events v1.8.0-rc.2
// replace github.com/containerd/stargz-snapshotter/ipfs/client => github.com/apostasie/stargz-snapshotter/ipfs/client 7014809d9d1721e2c6eb476ccd8f046bbe39e21d

// Pending https://github.com/data-accelerator/zdfs/pull/3
// replace github.com/data-accelerator/zdfs => github.com/apostasie/zdfs v0.0.0-20240624193050-cb46fc0a8f42

// Pending https://github.com/containerd/accelerated-container-image/pull/290
replace github.com/containerd/accelerated-container-image => github.com/apostasie/accelerated-container-image v0.0.0-20240624220112-f830841902e0

// Pending https://github.com/containerd/stargz-snapshotter/pull/1722
replace github.com/containerd/stargz-snapshotter => github.com/apostasie/stargz-snapshotter v0.0.0-20240624071742-7014809d9d17
replace github.com/containerd/stargz-snapshotter/ipfs => github.com/apostasie/stargz-snapshotter/ipfs v0.0.0-20240624071742-7014809d9d17
replace github.com/containerd/stargz-snapshotter/estargz => github.com/apostasie/stargz-snapshotter/estargz v0.0.0-20240624071742-7014809d9d17

// Pending moby updating to containerd v2 - no current PR AFAIK
// Unfortunately, pkg/sysinfo is very popular and used a large variety of places in our dependencies, so forking internally
// and forcing use
replace github.com/docker/docker/pkg/sysinfo => github.com/apostasie/nerdctl/v2/pkg/sysinfo c7d0714ad5045a180cfc59d74564a3c861eb8068
replace github.com/apostasie/nerdctl/v2/pkg/sysinfo => ./pkg/sysinfo

// XXXX
replace github.com/containerd/containerd v1.7.0 => github.com/containerd/containerd/api v1.8.0-rc.2
replace github.com/containerd/containerd v1.7.18 => github.com/containerd/containerd/api v1.8.0-rc.2

require (
	github.com/docker/docker/pkg/sysinfo v0.0.0
	// c7d0714ad5045a180cfc59d74564a3c861eb8068
//	github.com/containerd/nerdctl/v2/pkg/sysinfo v0.0.0-00010101000000-000000000000

	github.com/containerd/containerd/api v1.8.0-rc.2
	github.com/containerd/containerd/v2 v2.0.0-rc.3
	github.com/containerd/accelerated-container-image v0.0.0-00010101000000-000000000000
	github.com/containerd/stargz-snapshotter v0.0.0-00010101000000-000000000000
	github.com/containerd/stargz-snapshotter/estargz v0.15.1
	github.com/containerd/stargz-snapshotter/ipfs v0.0.0-00010101000000-000000000000
	github.com/containerd/imgcrypt v1.1.12-0.20240528203804-3ca09a2db5cd

	github.com/Masterminds/semver/v3 v3.2.1
	github.com/Microsoft/go-winio v0.6.2
	github.com/Microsoft/hcsshim v0.12.4
	github.com/compose-spec/compose-go/v2 v2.1.3
	github.com/containerd/cgroups/v3 v3.0.3
	github.com/containerd/console v1.0.4
	github.com/containerd/continuity v0.4.3
	github.com/containerd/errdefs v0.1.0
	github.com/containerd/fifo v1.1.0
	github.com/containerd/go-cni v1.1.9
	github.com/containerd/log v0.1.0
	github.com/containerd/platforms v0.2.1
	github.com/containerd/typeurl/v2 v2.1.1
	github.com/containernetworking/cni v1.2.0
	github.com/containernetworking/plugins v1.4.1
	github.com/coreos/go-iptables v0.7.0
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/cyphar/filepath-securejoin v0.2.3
	github.com/distribution/reference v0.6.0
	github.com/docker/cli v26.1.4+incompatible
	github.com/docker/docker v24.0.9+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.5.0
	github.com/fahedouch/go-logrotate v0.2.1
	github.com/fatih/color v1.15.0
	github.com/fluent/fluent-logger-golang v1.9.0
	github.com/ipfs/go-cid v0.0.7
	github.com/klauspost/compress v1.17.9
	github.com/mattn/go-isatty v0.0.17
	github.com/mitchellh/mapstructure v1.5.0
	github.com/moby/sys/mount v0.3.3
	github.com/moby/sys/signal v0.7.0
	github.com/moby/term v0.5.0
	github.com/muesli/cancelreader v0.2.2
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0
	github.com/opencontainers/runtime-spec v1.2.0
	github.com/pelletier/go-toml/v2 v2.2.2
	github.com/rootless-containers/bypass4netns v0.4.1
	github.com/rootless-containers/rootlesskit/v2 v2.1.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/tidwall/gjson v1.17.1
	github.com/vishvananda/netlink v1.2.1-beta.2
	github.com/vishvananda/netns v0.0.4
	github.com/yuchanns/srslog v1.1.0
	go.uber.org/mock v0.4.0
	golang.org/x/crypto v0.23.0
	golang.org/x/net v0.25.0
	golang.org/x/sync v0.7.0
	golang.org/x/sys v0.21.0
	golang.org/x/term v0.20.0
	golang.org/x/text v0.15.0
	gopkg.in/yaml.v3 v3.0.1
	gotest.tools/v3 v3.5.1
)

require (
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230811130428-ced1acdcaa24 // indirect
	github.com/AdamKorcz/go-118-fuzz-build v0.0.0-20230306123547-8075edf89bb0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/cilium/ebpf v0.11.0 // indirect
	github.com/containerd/go-runc v1.1.0 // indirect
	github.com/containerd/plugin v0.1.0 // indirect
	github.com/containerd/ttrpc v1.2.4 // indirect
	github.com/containers/ocicrypt v1.1.10 // indirect
	github.com/djherbis/times v1.6.0 // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.0.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-shellwords v1.0.12 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/sys/mountinfo v0.7.1 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/symlink v0.2.0 // indirect
	github.com/moby/sys/user v0.1.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multiaddr v0.12.4 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/opencontainers/runtime-tools v0.9.1-0.20221107090550-2e043c6bd626 // indirect
	github.com/opencontainers/selinux v1.11.0 // indirect
	github.com/philhofer/fwd v1.1.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/stefanberger/go-pkcs11uri v0.0.0-20201008174630-78d3cae3a980 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tinylib/msgp v1.1.9 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	go.mozilla.org/pkcs7 v0.0.0-20200128120323-432b2356ecb1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.51.0 // indirect
	go.opentelemetry.io/otel v1.26.0 // indirect
	go.opentelemetry.io/otel/metric v1.26.0 // indirect
	go.opentelemetry.io/otel/trace v1.26.0 // indirect
	golang.org/x/exp v0.0.0-20240112132812-db7319d0e0e3 // indirect
	golang.org/x/mod v0.18.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240415180920-8c6c420018be // indirect
	google.golang.org/grpc v1.63.2 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	lukechampine.com/blake3 v1.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
	tags.cncf.io/container-device-interface v0.7.2 // indirect
	tags.cncf.io/container-device-interface/specs-go v0.7.0 // indirect
)

//	github.com/docker/docker v24.0.9+incompatible
//	github.com/docker/docker v26.1.4+incompatible
// replace github.com/moby/moby/pkg/sysinfo  => ./pkg/sysinfo
