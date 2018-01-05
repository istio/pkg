package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "bag.go",
        "dictState.go",
        "emptyBag.go",
        "list.gen.go",
        "mutableBag.go",
        "protoBag.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//mixer/pkg/pool:go_default_library",
        "//pkg/log:go_default_library",
        "@com_github_hashicorp_go_multierror//:go_default_library",
        "@io_istio_api//mixer/v1:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["bag_test.go"],
    library = ":go_default_library",
    deps = [
        "@io_istio_api//mixer/v1:go_default_library",
        "@org_uber_go_zap//zapcore:go_default_library",
    ],
)
