// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

module github.com/open-edge-platform/virtual-edge-node/edge-node-simulator

go 1.24.6

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.9-20250912141014-52f32327d4b0.1
	github.com/Nerzal/gocloak/v13 v13.9.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/go-co-op/gocron v1.37.0
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/uuid v1.6.0
	github.com/magefile/mage v1.15.0
	github.com/manifoldco/promptui v0.9.0
	github.com/onsi/ginkgo/v2 v2.25.2
	github.com/onsi/gomega v1.38.2
	github.com/open-edge-platform/infra-core/apiv2/v2 v2.5.6
	github.com/open-edge-platform/infra-core/inventory/v2 v2.30.10
	github.com/open-edge-platform/infra-managers/host v1.23.0
	github.com/open-edge-platform/infra-managers/maintenance v1.23.3
	github.com/open-edge-platform/infra-managers/telemetry v1.23.0
	github.com/open-edge-platform/infra-onboarding/onboarding-manager v1.38.6
	github.com/open-edge-platform/orch-library/go v0.6.1
	github.com/stretchr/testify v1.11.1
	golang.org/x/oauth2 v0.30.0
	golang.org/x/term v0.35.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250825161204-c5933d9347a5
	google.golang.org/grpc v1.75.1
	google.golang.org/protobuf v1.36.10
	gopkg.in/yaml.v3 v3.0.1
	gotest.tools v2.2.0+incompatible
)

require (
	ariga.io/atlas v0.37.0 // indirect
	buf.build/go/protovalidate v1.0.0 // indirect
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	entgo.io/contrib v0.7.0 // indirect
	entgo.io/ent v0.14.5 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/agnivade/levenshtein v1.2.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/getkin/kin-openapi v0.132.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/inflect v0.21.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/go-resty/resty/v2 v2.16.5 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/cel-go v0.26.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus v1.1.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.2.0 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-7 // indirect
	github.com/hashicorp/hcl/v2 v2.23.0 // indirect
	github.com/hashicorp/vault/api v1.21.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.6 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/labstack/echo-contrib v0.17.4 // indirect
	github.com/labstack/echo/v4 v4.13.4 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/lestrrat-go/blackmagic v1.0.4 // indirect
	github.com/lestrrat-go/dsig v1.0.0 // indirect
	github.com/lestrrat-go/dsig-secp256k1 v1.0.0 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc/v3 v3.0.1 // indirect
	github.com/lestrrat-go/jwx/v3 v3.0.11 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/lestrrat-go/option/v2 v2.0.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mennanov/fmutils v0.3.3 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oapi-codegen/runtime v1.1.1 // indirect
	github.com/oasdiff/yaml v0.0.0-20250309154309-f31be36b4037 // indirect
	github.com/oasdiff/yaml3 v0.0.0-20250309153720-d2182401db90 // indirect
	github.com/open-edge-platform/orch-library/go/dazl v0.5.4 // indirect
	github.com/open-policy-agent/opa v1.9.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	github.com/sirupsen/logrus v1.9.4-0.20230606125235-dd1b4c2e81af // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/tchap/go-patricia/v2 v2.3.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/vektah/gqlparser/v2 v2.5.30 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/yashtewari/glob-intersection v0.2.0 // indirect
	github.com/zclconf/go-cty v1.16.3 // indirect
	github.com/zclconf/go-cty-yaml v1.1.0 // indirect
	go.einride.tech/aip v0.73.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.42.0 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/mod v0.28.0 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	golang.org/x/time v0.13.0 // indirect
	golang.org/x/tools v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250825161204-c5933d9347a5 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
